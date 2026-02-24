import { useNavigate } from "react-router-dom";
import { CreateOrganization, useOrganization } from "@clerk/clerk-react";
import { OnboardingForm } from "@/components/onboarding/OnboardingForm";
import { useOnboardingGuard } from "@/hooks/useOnboardingGuard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function OnboardingPage() {
  const navigate = useNavigate();
  const { organization } = useOrganization();
  const { hasOrganization, isLoading } = useOnboardingGuard();

  const handleComplete = () => {
    navigate("/");
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  if (!hasOrganization || !organization) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gray-50">
        <div className="w-full max-w-md">
          <Card>
            <CardHeader className="text-center">
              <CardTitle>Create Your Organization</CardTitle>
              <CardDescription>
                Let's set up your organization to get started with InfraLayer
              </CardDescription>
            </CardHeader>
            <CardContent>
              <CreateOrganization
                afterCreateOrganizationUrl="/onboarding"
                appearance={{
                  elements: {
                    rootBox: "w-full",
                    cardBox: "shadow-none border-0",
                  },
                }}
                hideSlug={true}
              />
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-2xl mx-auto px-4">
        <OnboardingForm onComplete={handleComplete} />
      </div>
    </div>
  );
}
