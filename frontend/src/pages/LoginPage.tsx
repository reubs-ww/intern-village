import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { LoginButton } from '@/components/auth/LoginButton'
import { APP_NAME } from '@/lib/constants'

export function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary text-2xl font-bold text-primary-foreground">
            IV
          </div>
          <CardTitle className="text-2xl">{APP_NAME}</CardTitle>
          <CardDescription>
            AI-powered development assistant that breaks down tasks into manageable subtasks
          </CardDescription>
        </CardHeader>
        <CardContent>
          <LoginButton className="w-full" />
        </CardContent>
      </Card>
    </div>
  )
}
