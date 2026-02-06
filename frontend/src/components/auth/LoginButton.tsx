import { Github } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface LoginButtonProps {
  className?: string
}

export function LoginButton({ className }: LoginButtonProps) {
  const handleLogin = () => {
    window.location.href = '/api/auth/github'
  }

  return (
    <Button className={className} size="lg" onClick={handleLogin}>
      <Github className="mr-2 h-5 w-5" />
      Sign in with GitHub
    </Button>
  )
}
