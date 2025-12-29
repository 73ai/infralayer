import { useEffect, useState, useRef } from "react";
import { useAuth, useUser } from "@clerk/clerk-react";
import { autorun } from "mobx";
import { useApiClient } from "@/lib/api";
import { userStore } from "@/stores/UserStore";

export interface OnboardingStatus {
  isLoading: boolean;
  isComplete: boolean;
  hasOrganization: boolean;
  error: string | null;
}

export const useOnboardingGuard = (): OnboardingStatus => {
  const [status, setStatus] = useState<OnboardingStatus>({
    isLoading: true,
    isComplete: false,
    hasOrganization: false,
    error: null,
  });

  const { isSignedIn, isLoaded: authLoaded, orgId: authOrgId } = useAuth();
  const { user } = useUser();
  const { getMe } = useApiClient();

  const clerkUserId = user?.id;

  const loadedForOrgRef = useRef<string | null>(null);

  useEffect(() => {
    const checkOnboardingStatus = async () => {
      if (!authLoaded) return;

      if (!isSignedIn) {
        setStatus({
          isLoading: false,
          isComplete: false,
          hasOrganization: false,
          error: null,
        });
        return;
      }

      if (!authOrgId) {
        setStatus({
          isLoading: false,
          isComplete: false,
          hasOrganization: false,
          error: null,
        });
        return;
      }

      if (!clerkUserId) return;

      try {
        if (userStore.loading) return;

        if (!userStore.userProfile || loadedForOrgRef.current !== authOrgId) {
          await userStore.loadUserProfile(getMe, clerkUserId, authOrgId);
          loadedForOrgRef.current = authOrgId;
        }

        const isComplete = Boolean(userStore.userProfile?.metadata?.completed_at);

        setStatus({
          isLoading: false,
          isComplete,
          hasOrganization: true,
          error: null,
        });
      } catch {
        setStatus({
          isLoading: false,
          isComplete: false,
          hasOrganization: true,
          error: null,
        });
      }
    };

    const dispose = autorun(() => {
      void userStore.loading;
      void userStore.userProfile;
      checkOnboardingStatus();
    });

    return () => dispose();
  }, [
    isSignedIn,
    authLoaded,
    authOrgId,
    getMe,
    clerkUserId,
  ]);

  return status;
};
