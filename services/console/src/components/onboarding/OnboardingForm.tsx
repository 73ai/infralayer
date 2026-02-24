import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { observer } from "mobx-react-lite";
import { useUser, useOrganization } from "@clerk/clerk-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form } from "@/components/ui/form";
import { CompanySizeStep } from "./steps/CompanySizeStep";
import { UseCasesStep } from "./steps/UseCasesStep";
import { ObservabilityStackStep } from "./steps/ObservabilityStackStep";
import { SummaryStep } from "./steps/SummaryStep";
import { useApiClient } from "@/lib/api";
import { userStore } from "@/stores/UserStore";
import { toast } from "sonner";
import {
  COMPANY_SIZES,
  TEAM_SIZES,
  USE_CASES,
  OBSERVABILITY_STACK,
} from "@/lib/onboarding-constants";
import type { OnboardingFormData } from "@/lib/onboarding-constants";

const onboardingSchema = z.object({
  companySize: z.enum(
    COMPANY_SIZES.map((c) => c.value) as [string, ...string[]],
  ),
  teamSize: z.enum(TEAM_SIZES.map((t) => t.value) as [string, ...string[]]),
  useCases: z
    .array(z.enum(USE_CASES.map((u) => u.value) as [string, ...string[]]))
    .min(1, "Please select at least one use case"),
  observabilityStack: z
    .array(
      z.enum(OBSERVABILITY_STACK.map((o) => o.value) as [string, ...string[]]),
    )
    .min(1, "Please select at least one tool"),
});

interface OnboardingFormProps {
  onComplete: () => void;
}

const STEPS = [
  { id: 1, title: "Company Info", description: "Tell us about your company" },
  { id: 2, title: "Use Cases", description: "What do you want to monitor?" },
  {
    id: 3,
    title: "Current Stack",
    description: "What tools do you currently use?",
  },
  { id: 4, title: "Summary", description: "Review your information" },
];

export const OnboardingForm = observer(
  ({ onComplete }: OnboardingFormProps) => {
    const [currentStep, setCurrentStep] = useState(1);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [isLoadingData, setIsLoadingData] = useState(true);
    const { user } = useUser();
    const { organization } = useOrganization();
    const { getMe, setOrganizationMetadata } = useApiClient();

    const clerkUserId = user?.id;
    const clerkOrgId = organization?.id;

    const form = useForm<OnboardingFormData>({
      resolver: zodResolver(onboardingSchema),
      defaultValues: {
        companySize: undefined,
        teamSize: undefined,
        useCases: [],
        observabilityStack: [],
      },
    });

    // Load user profile and prefill existing data
    useEffect(() => {
      const loadExistingData = async () => {
        try {
          // Load user profile if not already loaded
          if (
            !userStore.userProfile &&
            !userStore.loading &&
            clerkUserId &&
            clerkOrgId
          ) {
            await userStore.loadUserProfile(getMe, clerkUserId, clerkOrgId);
          }

          // If metadata exists, prefill the form
          if (
            userStore.userProfile?.metadata &&
            userStore.userProfile.metadata.company_size
          ) {
            form.reset({
              companySize: userStore.userProfile.metadata
                .company_size as OnboardingFormData["companySize"],
              teamSize: userStore.userProfile.metadata
                .team_size as OnboardingFormData["teamSize"],
              useCases:
                (userStore.userProfile.metadata
                  .use_cases as OnboardingFormData["useCases"]) || [],
              observabilityStack:
                (userStore.userProfile.metadata
                  .observability_stack as OnboardingFormData["observabilityStack"]) ||
                [],
            });
          }
        } catch (error) {
          // If user profile loading fails, that's fine - form stays with defaults
          console.error("Failed to load user profile for onboarding:", error);
        } finally {
          setIsLoadingData(false);
        }
      };

      loadExistingData();
    }, [getMe, form, clerkUserId, clerkOrgId]);

    const nextStep = async () => {
      let fieldsToValidate: (keyof OnboardingFormData)[] = [];

      // Validate only the fields for the current step
      switch (currentStep) {
        case 1:
          fieldsToValidate = ["companySize", "teamSize"];
          break;
        case 2:
          fieldsToValidate = ["useCases"];
          break;
        case 3:
          fieldsToValidate = ["observabilityStack"];
          break;
        default:
          fieldsToValidate = [];
      }

      const isValid =
        fieldsToValidate.length === 0 || (await form.trigger(fieldsToValidate));

      if (isValid) {
        setCurrentStep((prev) => Math.min(prev + 1, STEPS.length));
      }
    };

    const prevStep = () => {
      setCurrentStep((prev) => Math.max(prev - 1, 1));
    };

    const handleFinalSubmit = async () => {
      if (!userStore.organizationId) {
        toast.error("Organization not found");
        return;
      }

      // Validate the form first
      const isValid = await form.trigger();
      if (!isValid) {
        toast.error("Please fill in all required fields");
        return;
      }

      // Get form data
      const data = form.getValues();

      setIsSubmitting(true);
      try {
        // Use the organization ID from userStore
        if (!userStore.organizationId) {
          throw new Error("Organization ID not available");
        }

        await setOrganizationMetadata({
          organization_id: userStore.organizationId,
          company_size: data.companySize!,
          team_size: data.teamSize!,
          use_cases: data.useCases,
          observability_stack: data.observabilityStack,
        });

        // Update the userStore with the new metadata
        userStore.updateMetadata({
          company_size: data.companySize!,
          team_size: data.teamSize!,
          use_cases: data.useCases,
          observability_stack: data.observabilityStack,
          completed_at: new Date().toISOString(),
        });

        toast.success("Onboarding completed successfully!");
        onComplete();
      } catch (error) {
        toast.error("Failed to save onboarding information");
        console.error("Onboarding submission error:", error);
      } finally {
        setIsSubmitting(false);
      }
    };

    const renderStep = () => {
      switch (currentStep) {
        case 1:
          return <CompanySizeStep form={form} />;
        case 2:
          return <UseCasesStep form={form} />;
        case 3:
          return <ObservabilityStackStep form={form} />;
        case 4:
          return <SummaryStep form={form} />;
        default:
          return null;
      }
    };

    // Show loading while fetching existing data
    if (isLoadingData) {
      return (
        <div className="min-h-screen flex items-center justify-center p-4">
          <Card className="w-full max-w-2xl">
            <CardContent className="p-8">
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
                <p className="text-muted-foreground">
                  Loading your information...
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      );
    }

    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <Card className="w-full max-w-2xl">
          <CardHeader>
            <div className="flex items-center justify-between mb-4">
              <div>
                <CardTitle>Welcome to InfraLayer</CardTitle>
                <CardDescription>
                  Let's set up your organization profile to get started
                </CardDescription>
              </div>
              <div className="text-sm text-muted-foreground">
                Step {currentStep} of {STEPS.length}
              </div>
            </div>

            {/* Progress bar */}
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className="bg-primary h-2 rounded-full transition-all duration-300"
                style={{ width: `${(currentStep / STEPS.length) * 100}%` }}
              />
            </div>

            <div className="mt-4">
              <h3 className="font-semibold">{STEPS[currentStep - 1].title}</h3>
              <p className="text-sm text-muted-foreground">
                {STEPS[currentStep - 1].description}
              </p>
            </div>
          </CardHeader>

          <CardContent>
            <Form {...form}>
              <div className="space-y-6">
                {renderStep()}

                <div className="flex justify-between pt-6">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={prevStep}
                    disabled={currentStep === 1}
                  >
                    Previous
                  </Button>

                  {currentStep === STEPS.length ? (
                    <Button
                      type="button"
                      onClick={handleFinalSubmit}
                      disabled={isSubmitting}
                    >
                      {isSubmitting ? "Completing..." : "Complete Setup"}
                    </Button>
                  ) : (
                    <Button type="button" onClick={nextStep}>
                      Next
                    </Button>
                  )}
                </div>
              </div>
            </Form>
          </CardContent>
        </Card>
      </div>
    );
  },
);
