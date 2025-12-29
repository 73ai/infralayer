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
  const retryStateRef = useRef<{
    orgId: string | null;
    failCount: number;
    lastAttempt: number;
  }>({ orgId: null, failCount: 0, lastAttempt: 0 });

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

        if (retryStateRef.current.orgId !== authOrgId) {
          retryStateRef.current = { orgId: authOrgId, failCount: 0, lastAttempt: 0 };
        }

        // Exponential backoff for retries
        if (retryStateRef.current.failCount > 0) {
          const backoffMs = Math.min(1000 * Math.pow(2, retryStateRef.current.failCount - 1), 10000);
          const timeSinceLastAttempt = Date.now() - retryStateRef.current.lastAttempt;
          if (timeSinceLastAttempt < backoffMs) {
            setTimeout(() => {
              void userStore.loading;
            }, backoffMs - timeSinceLastAttempt);
            return;
          }
        }

        if (!userStore.userProfile || loadedForOrgRef.current !== authOrgId) {
          retryStateRef.current.lastAttempt = Date.now();
          await userStore.loadUserProfile(getMe, clerkUserId, authOrgId);
          loadedForOrgRef.current = authOrgId;
          retryStateRef.current.failCount = 0;
        }

        const isComplete = Boolean(userStore.userProfile?.metadata?.completed_at);

        setStatus({
          isLoading: false,
          isComplete,
          hasOrganization: true,
          error: null,
        });
      } catch {
        retryStateRef.current.failCount++;
        retryStateRef.current.lastAttempt = Date.now();

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
