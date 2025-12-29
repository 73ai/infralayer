import { makeAutoObservable, runInAction } from "mobx";

export interface UserProfile {
  id: string;
  name: string;
  slug: string;
  created_at: string;
  organization_id: string;
  user_id: string;
  metadata: {
    company_size: string;
    team_size: string;
    use_cases: string[];
    observability_stack: string[];
    completed_at: string;
  };
}

class UserStore {
  userProfile: UserProfile | null = null;
  loading = false;
  error: string | null = null;

  constructor() {
    makeAutoObservable(this);
  }

  get organizationId(): string | null {
    return this.userProfile?.organization_id || null;
  }

  get userId(): string | null {
    return this.userProfile?.user_id || null;
  }

  get isLoaded(): boolean {
    return this.userProfile !== null;
  }

  get hasCompletedOnboarding(): boolean {
    return Boolean(this.userProfile?.metadata?.completed_at);
  }

  async loadUserProfile(
    getMeFunction: (
      clerkUserId: string,
      clerkOrgId: string,
    ) => Promise<UserProfile>,
    clerkUserId: string,
    clerkOrgId: string,
  ): Promise<void> {
    if (this.loading) return; // Prevent duplicate requests

    runInAction(() => {
      this.loading = true;
      this.error = null;
    });

    try {
      const userProfile = await getMeFunction(clerkUserId, clerkOrgId);

      runInAction(() => {
        this.userProfile = userProfile;
        this.loading = false;
      });
    } catch (error) {
      runInAction(() => {
        this.error =
          error instanceof Error
            ? error.message
            : "Failed to load user profile";
        this.loading = false;
      });
      throw error;
    }
  }

  updateMetadata(metadata: Partial<UserProfile["metadata"]>): void {
    if (!this.userProfile) return;

    runInAction(() => {
      this.userProfile!.metadata = {
        ...this.userProfile!.metadata,
        ...metadata,
      };
    });
  }

  clearError(): void {
    this.error = null;
  }

  reset(): void {
    runInAction(() => {
      this.userProfile = null;
      this.loading = false;
      this.error = null;
    });
  }
}

// Export singleton instance
export const userStore = new UserStore();

// Export class for testing
export { UserStore };
