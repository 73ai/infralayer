import { useState } from "react";
import { useAuth, useOrganization, useUser } from "@clerk/clerk-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { CheckCircle, XCircle, Loader2, Terminal } from "lucide-react";
import { deviceService, getDeviceErrorMessage } from "@/services/deviceService";
import { userStore } from "@/stores/UserStore";

type VerifyState = "input" | "verifying" | "success" | "error";

export default function CLIVerifyPage() {
  const { getToken } = useAuth();
  const { organization } = useOrganization();
  const { user } = useUser();

  const [userCode, setUserCode] = useState("");
  const [state, setState] = useState<VerifyState>("input");
  const [errorMessage, setErrorMessage] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!userCode.trim()) {
      setErrorMessage("Please enter the verification code");
      setState("error");
      return;
    }

    if (!organization?.id || !userStore.organizationId || !userStore.userId) {
      setErrorMessage("Organization or user information not available");
      setState("error");
      return;
    }

    setState("verifying");
    setErrorMessage("");

    try {
      const authToken = await getToken();
      if (!authToken) {
        throw new Error("Unable to get authentication token");
      }

      const response = await deviceService.authorizeDevice(
        userCode.toUpperCase().replace(/[^A-Z0-9-]/g, ""),
        userStore.organizationId,
        userStore.userId,
        authToken,
      );

      if (response.success) {
        setState("success");
      } else {
        setErrorMessage(response.error || "Verification failed");
        setState("error");
      }
    } catch (error) {
      setErrorMessage(getDeviceErrorMessage(error));
      setState("error");
    }
  };

  const handleReset = () => {
    setState("input");
    setUserCode("");
    setErrorMessage("");
  };

  const formatUserCode = (value: string) => {
    // Remove non-alphanumeric characters and convert to uppercase
    const cleaned = value.toUpperCase().replace(/[^A-Z0-9]/g, "");
    // Add hyphen after 4 characters if needed
    if (cleaned.length > 4) {
      return cleaned.slice(0, 4) + "-" + cleaned.slice(4, 8);
    }
    return cleaned;
  };

  const handleCodeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const formatted = formatUserCode(e.target.value);
    setUserCode(formatted);
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
            <Terminal className="h-6 w-6 text-primary" />
          </div>
          <CardTitle>Authorize CLI</CardTitle>
          <CardDescription>
            Enter the verification code displayed in your terminal to authorize the InfraLayer CLI.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {state === "input" && (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="userCode" className="text-sm font-medium">
                  Verification Code
                </label>
                <Input
                  id="userCode"
                  type="text"
                  value={userCode}
                  onChange={handleCodeChange}
                  placeholder="XXXX-XXXX"
                  className="text-center text-2xl font-mono tracking-wider"
                  maxLength={9}
                  autoFocus
                  autoComplete="off"
                />
                <p className="text-xs text-muted-foreground">
                  The code is displayed when you run <code className="bg-muted px-1 py-0.5 rounded">infralayer auth login</code>
                </p>
              </div>
              <Button type="submit" className="w-full" disabled={userCode.length < 9}>
                Verify Code
              </Button>
            </form>
          )}

          {state === "verifying" && (
            <div className="py-8 text-center">
              <Loader2 className="h-12 w-12 text-primary animate-spin mx-auto mb-4" />
              <p className="text-muted-foreground">Verifying code...</p>
            </div>
          )}

          {state === "success" && (
            <div className="py-8 text-center">
              <CheckCircle className="h-12 w-12 text-green-600 mx-auto mb-4" />
              <h3 className="text-lg font-semibold text-green-800 mb-2">
                CLI Authorized!
              </h3>
              <p className="text-muted-foreground mb-4">
                Your CLI has been successfully authorized. You can now return to your terminal.
              </p>
              <p className="text-sm text-muted-foreground">
                Authorized for: <strong>{organization?.name}</strong>
              </p>
            </div>
          )}

          {state === "error" && (
            <div className="py-8 text-center">
              <XCircle className="h-12 w-12 text-red-600 mx-auto mb-4" />
              <h3 className="text-lg font-semibold text-red-800 mb-2">
                Verification Failed
              </h3>
              <p className="text-muted-foreground mb-4">{errorMessage}</p>
              <Button onClick={handleReset} variant="outline" className="w-full">
                Try Again
              </Button>
            </div>
          )}

          {/* Organization context */}
          {organization && state !== "success" && (
            <div className="mt-6 pt-4 border-t">
              <p className="text-xs text-center text-muted-foreground">
                Authorizing for: <strong>{organization.name}</strong>
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
