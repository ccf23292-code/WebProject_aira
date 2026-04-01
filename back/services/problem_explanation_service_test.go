package services

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"warehouse-web/models"
)

func newExplanationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:explanations-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	ddl := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			username TEXT,
			email TEXT,
			password_hash TEXT,
			role TEXT,
			remember_token TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE user_profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			nickname TEXT,
			avatar_url TEXT,
			level INTEGER,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX idx_user_profiles_user_id ON user_profiles(user_id);`,
		`CREATE TABLE test_papers (
			id INTEGER PRIMARY KEY,
			course_id TEXT,
			name TEXT,
			created_at DATETIME
		);`,
		`CREATE TABLE problems (
			id INTEGER PRIMARY KEY,
			testpaper_id INTEGER,
			source_id TEXT,
			"order" INTEGER,
			sequence_id INTEGER,
			question_type TEXT,
			category TEXT,
			source_url TEXT,
			test TEXT,
			answer TEXT,
			score REAL,
			explanation TEXT,
			difficulty TEXT,
			options_json TEXT,
			tags_json TEXT
		);`,
		`CREATE TABLE problem_explanations (
			id INTEGER PRIMARY KEY,
			problem_id INTEGER,
			user_id INTEGER,
			content_md TEXT,
			up_votes INTEGER,
			down_votes INTEGER,
			score INTEGER,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX idx_problem_explanation ON problem_explanations(problem_id, user_id);`,
		`CREATE TABLE problem_explanation_votes (
			id INTEGER PRIMARY KEY,
			explanation_id INTEGER,
			user_id INTEGER,
			value INTEGER,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX idx_explanation_vote ON problem_explanation_votes(explanation_id, user_id);`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec ddl: %v", err)
		}
	}

	return db
}

func seedProblemForExplanationTests(t *testing.T, db *gorm.DB) models.Problem {
	t.Helper()

	paper := models.TestPaper{
		ID:        1,
		CourseID:  "CS1018F",
		Name:      "FDS Test Paper",
		CreatedAt: time.Now().UTC(),
	}
	if err := db.Create(&paper).Error; err != nil {
		t.Fatalf("create paper: %v", err)
	}

	problem := models.Problem{
		ID:          1001,
		TestpaperID: paper.ID,
		SourceID:    "p-1001",
		Order:       1,
		Test:        "sample problem",
		Answer:      "A",
	}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	return problem
}

func seedUserWithProfile(t *testing.T, db *gorm.DB, id uint64, username, nickname string) {
	t.Helper()

	user := models.User{
		ID:           id,
		Username:     username,
		Email:        username + "@zju.edu.cn",
		PasswordHash: "x",
		Role:         models.RoleStudent,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.UserProfile{
		UserID:    id,
		Nickname:  nickname,
		AvatarURL: "",
	}).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
}

func TestListProblemExplanationsReturnsTopThreeAndMine(t *testing.T) {
	db := newExplanationTestDB(t)
	problem := seedProblemForExplanationTests(t, db)

	seedUserWithProfile(t, db, 1, "alice", "Alice")
	seedUserWithProfile(t, db, 2, "bob", "Bob")
	seedUserWithProfile(t, db, 3, "carol", "Carol")
	seedUserWithProfile(t, db, 4, "dave", "Dave")

	explanations := []models.ProblemExplanation{
		{ID: 1, ProblemID: problem.ID, UserID: 1, ContentMD: "exp-1", UpVotes: 3, DownVotes: 0, Score: 3},
		{ID: 2, ProblemID: problem.ID, UserID: 2, ContentMD: "exp-2", UpVotes: 2, DownVotes: 0, Score: 2},
		{ID: 3, ProblemID: problem.ID, UserID: 3, ContentMD: "exp-3", UpVotes: 1, DownVotes: 0, Score: 1},
		{ID: 4, ProblemID: problem.ID, UserID: 4, ContentMD: "exp-4", UpVotes: 0, DownVotes: 0, Score: 0},
	}
	for _, explanation := range explanations {
		if err := db.Create(&explanation).Error; err != nil {
			t.Fatalf("create explanation: %v", err)
		}
	}

	service := NewProblemExplanationService(db, NewPaperService(db))
	viewerID := models.PrimaryKey(4)

	result, err := service.ListProblemExplanations(problem.ID, &viewerID)
	if err != nil {
		t.Fatalf("list explanations: %v", err)
	}

	if len(result.Items) != 3 {
		t.Fatalf("expected 3 top explanations, got %d", len(result.Items))
	}
	if result.Items[0].ID != 1 || result.Items[1].ID != 2 || result.Items[2].ID != 3 {
		t.Fatalf("unexpected ranking order: %#v", result.Items)
	}
	if result.MyItem == nil || result.MyItem.ID != 4 {
		t.Fatalf("expected my explanation outside top 3 to be returned separately: %#v", result.MyItem)
	}
	if result.MyItem.AuthorName != "Dave" {
		t.Fatalf("expected nickname to be used as author name, got %q", result.MyItem.AuthorName)
	}
}

func TestVoteProblemExplanationSwitchAndWithdraw(t *testing.T) {
	db := newExplanationTestDB(t)
	problem := seedProblemForExplanationTests(t, db)

	seedUserWithProfile(t, db, 1, "author", "Author")
	seedUserWithProfile(t, db, 2, "voter", "Voter")

	explanation := models.ProblemExplanation{
		ID:        11,
		ProblemID: problem.ID,
		UserID:    1,
		ContentMD: "my explanation",
	}
	if err := db.Create(&explanation).Error; err != nil {
		t.Fatalf("create explanation: %v", err)
	}

	service := NewProblemExplanationService(db, NewPaperService(db))

	if _, err := service.VoteProblemExplanation(problem.ID, explanation.ID, 2, 1); err != nil {
		t.Fatalf("upvote failed: %v", err)
	}
	var afterUp models.ProblemExplanation
	if err := db.First(&afterUp, explanation.ID).Error; err != nil {
		t.Fatalf("load after upvote: %v", err)
	}
	if afterUp.UpVotes != 1 || afterUp.DownVotes != 0 || afterUp.Score != 1 {
		t.Fatalf("unexpected counters after upvote: %#v", afterUp)
	}

	if _, err := service.VoteProblemExplanation(problem.ID, explanation.ID, 2, -1); err != nil {
		t.Fatalf("switch to downvote failed: %v", err)
	}
	var afterDown models.ProblemExplanation
	if err := db.First(&afterDown, explanation.ID).Error; err != nil {
		t.Fatalf("load after downvote: %v", err)
	}
	if afterDown.UpVotes != 0 || afterDown.DownVotes != 1 || afterDown.Score != -1 {
		t.Fatalf("unexpected counters after downvote: %#v", afterDown)
	}

	if _, err := service.VoteProblemExplanation(problem.ID, explanation.ID, 2, 0); err != nil {
		t.Fatalf("withdraw vote failed: %v", err)
	}
	var afterWithdraw models.ProblemExplanation
	if err := db.First(&afterWithdraw, explanation.ID).Error; err != nil {
		t.Fatalf("load after withdraw: %v", err)
	}
	if afterWithdraw.UpVotes != 0 || afterWithdraw.DownVotes != 0 || afterWithdraw.Score != 0 {
		t.Fatalf("unexpected counters after withdraw: %#v", afterWithdraw)
	}
}
