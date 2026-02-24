// GCP Integration Modal - Service Account JSON Configuration

import React, { useState, useCallback } from "react";
import { observer } from "mobx-react-lite";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../../components/ui/dialog";
import { Button } from "../../../components/ui/button";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "../../../components/ui/alert";
import { JSONEditor } from "../../../components/JSONEditor";
import {
  Loader2,
  AlertCircle,
  CheckCircle2,
  ExternalLink,
  Shield,
} from "lucide-react";
import { integrationStore } from "../../../stores/IntegrationStore";
import { userStore } from "../../../stores/UserStore";
import { useApiClient } from "../../../lib/api";

interface GCPIntegrationModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const GCPIntegrationModal: React.FC<GCPIntegrationModalProps> = observer(
  ({ isOpen, onClose }) => {
    const [serviceAccountJSON, setServiceAccountJSON] = useState("");
    const [isValidating, setIsValidating] = useState(false);
    const [isConnecting, setIsConnecting] = useState(false);
    const [isValidJSON, setIsValidJSON] = useState(false);
    const [validationResult, setValidationResult] = useState<{
      valid: boolean;
      errors?: string[];
    } | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [jsonError, setJsonError] = useState<string | null>(null);
    const { apiPost } = useApiClient();

    // Format JSON for display (currently unused but kept for future enhancement)
    // const formatJSON = (json: string): string => {
    //   try {
    //     const parsed = JSON.parse(json);
    //     return JSON.stringify(parsed, null, 2);
    //   } catch {
    //     return json;
    //   }
    // };

    // Handle JSON input change from CodeMirror
    const handleJSONChange = useCallback((value: string) => {
      setServiceAccountJSON(value);
      setError(null);
      setValidationResult(null);

      // Just check if it's valid JSON
      if (value.trim()) {
        try {
          JSON.parse(value);
          setIsValidJSON(true);
        } catch {
          setIsValidJSON(false);
        }
      } else {
        setIsValidJSON(false);
      }
    }, []);

    // Handle JSON validation errors from CodeMirror
    const handleJSONValidation = useCallback(
      (errors: string[] | null) => {
        if (errors && errors.length > 0) {
          setJsonError(errors[0]);
          setIsValidJSON(false);
        } else {
          setJsonError(null);
          // Re-check if valid JSON when no syntax errors
          if (serviceAccountJSON.trim()) {
            try {
              JSON.parse(serviceAccountJSON);
              setIsValidJSON(true);
            } catch {
              setIsValidJSON(false);
            }
          }
        }
      },
      [serviceAccountJSON],
    );

    // Validate credentials using the generic API
    const validateCredentials = useCallback(async () => {
      if (!isValidJSON || jsonError) {
        setError("Please provide valid JSON");
        return;
      }

      setIsValidating(true);
      setError(null);

      try {
        const response = await apiPost("/integrations/validate/", {
          connector_type: "gcp",
          credentials: {
            service_account_json: serviceAccountJSON,
          },
        });

        setValidationResult(response);

        const validationResponse = response as {
          valid: boolean;
          errors?: string[];
        };
        if (!validationResponse.valid) {
          const errorMessage =
            validationResponse.errors?.join(", ") || "Validation failed";
          setError(errorMessage);
        }
      } catch (err: unknown) {
        setError(
          err instanceof Error ? err.message : "Failed to validate credentials",
        );
      } finally {
        setIsValidating(false);
      }
    }, [isValidJSON, jsonError, serviceAccountJSON, apiPost]);

    // Connect the integration
    const handleConnect = useCallback(async () => {
      if (!validationResult?.valid) {
        setError("Please validate your credentials first");
        return;
      }

      if (!userStore.organizationId || !userStore.userId) {
        setError("Organization information not available");
        return;
      }

      setIsConnecting(true);
      setError(null);

      try {
        // Use the standard integration flow
        await integrationStore.handleCallback("gcp", {
          code: serviceAccountJSON,
          state: `${userStore.organizationId}:${userStore.userId}`,
        });

        // Reload integrations to show the new connection
        await integrationStore.loadIntegrations(userStore.organizationId);

        // Close modal on success
        onClose();
      } catch (err: unknown) {
        setError(
          err instanceof Error
            ? err.message
            : "Failed to connect GCP integration",
        );
      } finally {
        setIsConnecting(false);
      }
    }, [validationResult, serviceAccountJSON, onClose]);

    // Reset modal state when closed
    const handleClose = () => {
      setServiceAccountJSON("");
      setIsValidJSON(false);
      setValidationResult(null);
      setError(null);
      setJsonError(null);
      setIsValidating(false);
      setIsConnecting(false);
      onClose();
    };

    return (
      <Dialog open={isOpen} onOpenChange={handleClose}>
        <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <img
                src="/icons/gcp.svg"
                alt="Google Cloud Platform"
                className="w-6 h-6"
              />
              Connect Google Cloud Platform
            </DialogTitle>
            <DialogDescription>
              Configure GCP integration using a service account with Viewer role
              permissions.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {/* Requirements Alert */}
            <Alert>
              <Shield className="h-4 w-4" />
              <AlertTitle>Service Account Requirements</AlertTitle>
              <AlertDescription>
                <ul className="mt-2 space-y-1 text-sm">
                  <li>
                    • Service account must have the <strong>Viewer role</strong>{" "}
                    on your GCP project
                  </li>
                  <li>
                    • You can grant this role in the{" "}
                    <a
                      href="https://console.cloud.google.com/iam-admin/roles/details/roles%3Cviewer"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:underline inline-flex items-center gap-1"
                    >
                      IAM & Admin Console
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  </li>
                  <li>
                    • Download the service account key as JSON from the GCP
                    Console
                  </li>
                </ul>
              </AlertDescription>
            </Alert>

            {/* JSON Editor */}
            <div className="space-y-2">
              <label className="text-sm font-medium">
                Service Account JSON
              </label>
              <JSONEditor
                value={serviceAccountJSON}
                onChange={handleJSONChange}
                onValidation={handleJSONValidation}
                placeholder={`{
  "type": "service_account",
  "project_id": "your-project-id",
  "private_key_id": "...",
  "private_key": "-----BEGIN PRIVATE KEY-----\\n...\\n-----END PRIVATE KEY-----\\n",
  "client_email": "your-service-account@your-project.iam.gserviceaccount.com",
  ...
}`}
                height="350px"
                theme="dark"
              />
            </div>

            {/* Validation Result */}
            {validationResult && validationResult.valid && (
              <Alert className="border-green-200 bg-green-50">
                <CheckCircle2 className="h-4 w-4 text-green-600" />
                <AlertTitle className="text-green-800">
                  Credentials Validated
                </AlertTitle>
                <AlertDescription>
                  <div className="mt-2 text-sm text-green-700">
                    Your service account credentials have been validated
                    successfully. The service account has the necessary
                    permissions to integrate with GCP.
                  </div>
                </AlertDescription>
              </Alert>
            )}

            {/* Error Display */}
            {error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertTitle>Validation Error</AlertTitle>
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            {/* Security Notice */}
            <div className="rounded-lg bg-blue-50 p-3 text-sm">
              <div className="flex items-start space-x-2">
                <Shield className="h-4 w-4 text-blue-600 mt-0.5" />
                <div>
                  <p className="text-blue-800 font-medium">
                    Security Information
                  </p>
                  <p className="text-blue-700 mt-1">
                    Your service account credentials will be encrypted using
                    AES-256-GCM and stored securely. InfraLayer only accesses the
                    permissions explicitly granted to the service account.
                  </p>
                </div>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={handleClose}
              disabled={isValidating || isConnecting}
            >
              Cancel
            </Button>

            {!validationResult?.valid ? (
              <Button
                onClick={validateCredentials}
                disabled={isValidating || !isValidJSON || !!jsonError}
              >
                {isValidating ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Validating...
                  </>
                ) : (
                  "Validate Credentials"
                )}
              </Button>
            ) : (
              <Button
                onClick={handleConnect}
                disabled={isConnecting}
                className="bg-green-600 hover:bg-green-700"
              >
                {isConnecting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Connecting...
                  </>
                ) : (
                  "Connect GCP"
                )}
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  },
);

export default GCPIntegrationModal;
