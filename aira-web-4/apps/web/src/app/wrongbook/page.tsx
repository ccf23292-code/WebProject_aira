import { redirect } from 'next/navigation';

export default function WrongBookRedirect() {
  redirect('/profile/wrongbook');
}
