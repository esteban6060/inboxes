"use client";

import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { Card, CardContent } from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { ResetPasswordForm } from "./reset-password-form";

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={<Card><CardContent className="flex items-center justify-center p-12"><Spinner className="h-6 w-6" /></CardContent></Card>}>
      <ResetPasswordRoute />
    </Suspense>
  );
}

function ResetPasswordRoute() {
  const searchParams = useSearchParams();
  return <ResetPasswordForm token={searchParams.get("token") || ""} />;
}
