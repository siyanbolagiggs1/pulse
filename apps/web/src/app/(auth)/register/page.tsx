"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { GoogleLogin } from "@react-oauth/google";
import { Briefcase, TrendingUp } from "lucide-react";
import { authApi } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { toast } from "@/components/ui/use-toast";

const schema = z.object({
  name: z.string().min(2, "Name must be at least 2 characters"),
  email: z.string().email("Invalid email"),
  password: z.string().min(8, "Password must be at least 8 characters"),
  role: z.enum(["business", "promoter"]),
});
type FormData = z.infer<typeof schema>;

const roles = [
  {
    value: "business" as const,
    icon: Briefcase,
    title: "Business Owner",
    description: "Create adverts & run repost campaigns",
  },
  {
    value: "promoter" as const,
    icon: TrendingUp,
    title: "User / Promoter",
    description: "Earn money by promoting content",
  },
];

export default function RegisterPage() {
  const router = useRouter();
  const { setAuth } = useAuthStore();
  const [loading, setLoading] = useState(false);
  const [role, setRole] = useState<"business" | "promoter">("promoter");
  const [googleCredential, setGoogleCredential] = useState<string | null>(null);
  const [showRolePicker, setShowRolePicker] = useState(false);
  const [googleLoading, setGoogleLoading] = useState(false);

  const { register, handleSubmit, setValue, formState: { errors } } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { role: "promoter" },
  });

  const onSubmit = async (data: FormData) => {
    setLoading(true);
    try {
      const res = await authApi.register(data);
      setAuth(res.data.data.user, res.data.data.accessToken);
      toast({ title: "Account created!", description: "Check your email to verify your account." });
      router.push("/dashboard");
    } catch (err: any) {
      toast({ title: "Registration failed", description: err?.response?.data?.message ?? "Please try again", variant: "destructive" });
    } finally {
      setLoading(false);
    }
  };

  const handleGoogleSuccess = (credentialResponse: any) => {
    setGoogleCredential(credentialResponse.credential);
    setShowRolePicker(true);
  };

  const handleGoogleRoleSelect = async (selectedRole: "business" | "promoter") => {
    if (!googleCredential) return;
    setGoogleLoading(true);
    try {
      const res = await authApi.googleSignIn(googleCredential, selectedRole);
      setAuth(res.data.data.user, res.data.data.accessToken);
      setShowRolePicker(false);
      router.push("/dashboard");
    } catch (err: any) {
      toast({ title: "Google sign-up failed", description: err?.response?.data?.message, variant: "destructive" });
    } finally {
      setGoogleLoading(false);
    }
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Create account</CardTitle>
          <CardDescription>Join Pulse as a business or promoter</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-1">
              <Label htmlFor="name">Full name</Label>
              <Input id="name" placeholder="Jane Smith" {...register("name")} />
              {errors.name && <p className="text-xs text-destructive">{errors.name.message}</p>}
            </div>
            <div className="space-y-1">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" placeholder="you@example.com" {...register("email")} />
              {errors.email && <p className="text-xs text-destructive">{errors.email.message}</p>}
            </div>
            <div className="space-y-1">
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" placeholder="8+ characters" {...register("password")} />
              {errors.password && <p className="text-xs text-destructive">{errors.password.message}</p>}
            </div>

            <div className="space-y-2">
              <Label>I want to…</Label>
              <div className="grid grid-cols-2 gap-3">
                {roles.map(({ value, icon: Icon, title, description }) => (
                  <button
                    key={value}
                    type="button"
                    onClick={() => { setRole(value); setValue("role", value); }}
                    className={`flex flex-col items-start gap-1 rounded-lg border-2 bg-background p-3 text-left transition-colors ${
                      role === value
                        ? "border-primary bg-primary/5"
                        : "border-border hover:border-primary/50"
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      <Icon className={`h-4 w-4 ${role === value ? "text-primary" : "text-muted-foreground"}`} />
                      <span className={`text-sm font-medium ${role === value ? "text-primary" : ""}`}>{title}</span>
                    </div>
                    <span className="text-xs text-muted-foreground">{description}</span>
                  </button>
                ))}
              </div>
              {errors.role && <p className="text-xs text-destructive">{errors.role.message}</p>}
            </div>

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Creating account…" : "Create account"}
            </Button>
          </form>

          <div className="relative my-4">
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t border-border" />
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-card px-2 text-muted-foreground">or continue with Google</span>
            </div>
          </div>

          <div className="flex justify-center">
            <GoogleLogin
              onSuccess={handleGoogleSuccess}
              onError={() => toast({ title: "Google sign-up failed", variant: "destructive" })}
              theme="filled_black"
              shape="rectangular"
              width="100%"
            />
          </div>

          <p className="mt-4 text-center text-sm text-muted-foreground">
            Already have an account?{" "}
            <Link href="/login" className="text-primary hover:underline">Sign in</Link>
          </p>
        </CardContent>
      </Card>

      <Dialog open={showRolePicker} onOpenChange={setShowRolePicker}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>How will you use Pulse?</DialogTitle>
            <DialogDescription>Pick one to complete your sign-up</DialogDescription>
          </DialogHeader>
          <div className="grid grid-cols-1 gap-3 pt-2">
            {roles.map(({ value, icon: Icon, title, description }) => (
              <button
                key={value}
                type="button"
                disabled={googleLoading}
                onClick={() => handleGoogleRoleSelect(value)}
                className="flex items-start gap-4 rounded-lg border-2 border-border bg-background p-4 text-left transition-colors hover:border-primary hover:bg-primary/5 disabled:opacity-50"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10">
                  <Icon className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="font-medium">{title}</p>
                  <p className="text-sm text-muted-foreground">{description}</p>
                </div>
              </button>
            ))}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
