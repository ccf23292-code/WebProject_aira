// 共享类型定义

export enum ProblemDifficulty {
  EASY = 'easy',
  MEDIUM = 'medium',
  HARD = 'hard'
}

export enum SubmissionStatus {
  PENDING = 'pending',
  JUDGING = 'judging',
  ACCEPTED = 'accepted',
  WRONG_ANSWER = 'wrong_answer',
  TIME_LIMIT_EXCEEDED = 'time_limit_exceeded',
  MEMORY_LIMIT_EXCEEDED = 'memory_limit_exceeded',
  RUNTIME_ERROR = 'runtime_error',
  COMPILE_ERROR = 'compile_error',
  SYSTEM_ERROR = 'system_error'
}

export interface Problem {
  id: number;
  title: string;
  statement: string;
  inputFormat: string;
  outputFormat: string;
  samples: Array<{
    input: string;
    output: string;
    explanation?: string;
  }>;
  constraints: string;
  difficulty: ProblemDifficulty;
  tags: string[];
  timeLimit: number;
  memoryLimit: number;
  createdAt: Date;
  updatedAt: Date;
}

export interface Submission {
  id: number;
  problemId: number;
  userId?: number;
  username?: string;
  language: string;
  source: string;
  status: SubmissionStatus;
  time?: number;
  memory?: number;
  errorOutput?: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface CreateProblemDto {
  title: string;
  statement: string;
  inputFormat: string;
  outputFormat: string;
  samples: Array<{
    input: string;
    output: string;
    explanation?: string;
  }>;
  constraints: string;
  difficulty: ProblemDifficulty;
  tags: string[];
  timeLimit: number;
  memoryLimit: number;
}

export interface CreateSubmissionDto {
  problemId: number;
  language: string;
  source: string;
}

export interface ApiResponse<T> {
  data: T;
  meta?: {
    total?: number;
    page?: number;
    limit?: number;
  };
}

export interface ApiError {
  code: string;
  message: string;
  details?: any;
}