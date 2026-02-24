import { UseFormReturn } from "react-hook-form";
import {
  COMPANY_SIZES,
  TEAM_SIZES,
  USE_CASES,
  OBSERVABILITY_STACK,
} from "@/lib/onboarding-constants";
import type { OnboardingFormData } from "@/lib/onboarding-constants";

interface SummaryStepProps {
  form: UseFormReturn<OnboardingFormData>;
}

export function SummaryStep({ form }: SummaryStepProps) {
  const values = form.getValues();

  const getCompanySizeLabel = (value: string) =>
    COMPANY_SIZES.find((size) => size.value === value)?.label || value;

  const getTeamSizeLabel = (value: string) =>
    TEAM_SIZES.find((size) => size.value === value)?.label || value;

  const getUseCaseLabels = (values: string[]) =>
    values.map(
      (value) => USE_CASES.find((uc) => uc.value === value)?.label || value,
    );

  const getObservabilityStackLabels = (values: string[]) =>
    values.map(
      (value) =>
        OBSERVABILITY_STACK.find((tool) => tool.value === value)?.label ||
        value,
    );

  return (
    <div className="space-y-6">
      <div className="rounded-lg border p-4 space-y-4">
        <h3 className="font-semibold text-lg">Review Your Information</h3>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <h4 className="font-medium text-sm text-muted-foreground mb-2">
              Company Size
            </h4>
            <p className="text-sm">{getCompanySizeLabel(values.companySize)}</p>
          </div>

          <div>
            <h4 className="font-medium text-sm text-muted-foreground mb-2">
              Team Size
            </h4>
            <p className="text-sm">{getTeamSizeLabel(values.teamSize)}</p>
          </div>
        </div>

        <div>
          <h4 className="font-medium text-sm text-muted-foreground mb-2">
            Use Cases
          </h4>
          <div className="flex flex-wrap gap-2">
            {getUseCaseLabels(values.useCases).map((label, index) => (
              <span
                key={index}
                className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-primary/10 text-primary"
              >
                {label}
              </span>
            ))}
          </div>
        </div>

        <div>
          <h4 className="font-medium text-sm text-muted-foreground mb-2">
            Current Observability Stack
          </h4>
          <div className="flex flex-wrap gap-2">
            {getObservabilityStackLabels(values.observabilityStack).map(
              (label, index) => (
                <span
                  key={index}
                  className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-secondary text-secondary-foreground"
                >
                  {label}
                </span>
              ),
            )}
          </div>
        </div>
      </div>

      <div className="text-sm text-muted-foreground">
        Click "Complete Setup" to save your information and start using
        InfraLayer.
      </div>
    </div>
  );
}
