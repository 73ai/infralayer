import { UseFormReturn } from "react-hook-form";
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from "@/components/ui/form";
import { Checkbox } from "@/components/ui/checkbox";
import { USE_CASES } from "@/lib/onboarding-constants";
import type { OnboardingFormData } from "@/lib/onboarding-constants";

interface UseCasesStepProps {
  form: UseFormReturn<OnboardingFormData>;
}

export function UseCasesStep({ form }: UseCasesStepProps) {
  return (
    <div className="space-y-6">
      <FormField
        control={form.control}
        name="useCases"
        render={() => (
          <FormItem>
            <div className="mb-4">
              <FormLabel className="text-base">Use Cases</FormLabel>
              <FormDescription>
                Select all the areas where you want to use InfraLayer
              </FormDescription>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {USE_CASES.map((useCase) => (
                <FormField
                  key={useCase.value}
                  control={form.control}
                  name="useCases"
                  render={({ field }) => {
                    return (
                      <FormItem
                        key={useCase.value}
                        className="flex flex-row items-start space-x-3 space-y-0"
                      >
                        <FormControl>
                          <Checkbox
                            checked={field.value?.includes(useCase.value)}
                            onCheckedChange={(checked) => {
                              return checked
                                ? field.onChange([
                                    ...field.value,
                                    useCase.value,
                                  ])
                                : field.onChange(
                                    field.value?.filter(
                                      (value) => value !== useCase.value,
                                    ),
                                  );
                            }}
                          />
                        </FormControl>
                        <FormLabel className="text-sm font-normal cursor-pointer">
                          {useCase.label}
                        </FormLabel>
                      </FormItem>
                    );
                  }}
                />
              ))}
            </div>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  );
}
