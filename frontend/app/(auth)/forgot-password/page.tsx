"use client";

import { useState } from "react";
import Link from "next/link";
import { api, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { ResetPasswordForm } from "../reset-password/reset-password-form";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      await api.post("/api/auth/forgot-password", { email });
      setSent(true);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.message === "email_not_configured") {
          setError(
            "Password reset is unavailable. Email sending is not configured on this instance. Contact your administrator."
          );
        } else {
          setError(err.message);
        }
      } else {
        setError("Something went wrong");
      }
    } finally {
      setLoading(false);
    }
  }

  if (sent) {
    return <ResetPasswordForm email={email} />;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Reset your password</CardTitle>
        <CardDescription>
          Enter your email and we&apos;ll send you a reset code
        </CardDescription>
      </CardHeader>
      <form onSubmit={handleSubmit}>
        <CardContent className="space-y-4">
          {error && (
            <div role="alert" className="text-sm text-destructive bg-destructive/10 p-3 rounded-md">
              {error}
            </div>
          )}
          <div className="space-y-2">
            <label htmlFor="email" className="text-sm font-medium">
              Email
            </label>
            <Input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              required
            />
          </div>
        </CardContent>
        <CardFooter className="flex flex-col space-y-4">
          <Button className="w-full" disabled={loading}>
            {loading ? <Spinner className="mr-2" /> : null}
            Send reset code
          </Button>
          <Link href="/reset-password" className="text-sm text-muted-foreground hover:text-primary">
            I already have a code
          </Link>
          <Link
            href="/login"
            className="text-sm text-muted-foreground hover:text-primary"
          >
            Back to sign in
          </Link>
        </CardFooter>
      </form>
    </Card>
  );
}
