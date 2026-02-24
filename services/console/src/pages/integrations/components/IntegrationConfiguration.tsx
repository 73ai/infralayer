import React from "react";
import { Integration, Connector } from "../../../types/integration";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../components/ui/card";
import { Badge } from "../../../components/ui/badge";
import { Settings, Shield, Globe, Key } from "lucide-react";

interface IntegrationConfigurationProps {
  integration: Integration;
  connector: Connector;
}

export const IntegrationConfiguration: React.FC<
  IntegrationConfigurationProps
> = ({ integration, connector }) => {
  const config = integration.configuration;

  const renderAuthTypeInfo = () => {
    switch (connector.authType) {
      case "oauth2":
        return (
          <div className="flex items-center space-x-2 text-sm">
            <Shield className="h-4 w-4 text-green-600" />
            <span className="text-gray-600">OAuth 2.0 Authentication</span>
          </div>
        );
      case "app_installation":
        return (
          <div className="flex items-center space-x-2 text-sm">
            <Globe className="h-4 w-4 text-blue-600" />
            <span className="text-gray-600">App Installation</span>
          </div>
        );
      case "api_key":
        return (
          <div className="flex items-center space-x-2 text-sm">
            <Key className="h-4 w-4 text-purple-600" />
            <span className="text-gray-600">API Key Authentication</span>
          </div>
        );
      default:
        return null;
    }
  };

  const renderCapabilities = () => {
    return (
      <div>
        <h4 className="text-sm font-medium text-gray-900 mb-3">Capabilities</h4>
        <div className="flex flex-wrap gap-2">
          {connector.capabilities.map((capability) => (
            <Badge key={capability} variant="secondary" className="text-xs">
              {capability.replace(/_/g, " ")}
            </Badge>
          ))}
        </div>
      </div>
    );
  };

  const renderConnectorSpecificSettings = () => {
    if (!config) return null;

    switch (connector.type) {
      case "slack":
        return (
          <div className="space-y-4">
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-3">
                Bot Configuration
              </h4>
              <div className="bg-gray-50 rounded-lg p-4 space-y-3">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="text-gray-500">Bot User</span>
                    <p className="font-medium">@infralayer</p>
                  </div>
                  <div>
                    <span className="text-gray-500">Response Mode</span>
                    <p className="font-medium">Thread Replies</p>
                  </div>
                </div>
              </div>
            </div>

            {config.connectedChannels && (
              <div>
                <h4 className="text-sm font-medium text-gray-900 mb-3">
                  Channel Settings
                </h4>
                <div className="bg-gray-50 rounded-lg p-4">
                  <div className="text-sm">
                    <span className="text-gray-500">Active Channels: </span>
                    <span className="font-medium">
                      {config.connectedChannels.length}
                    </span>
                  </div>
                  {config.connectedChannels.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1">
                      {config.connectedChannels.slice(0, 8).map((channel) => (
                        <Badge
                          key={channel}
                          variant="outline"
                          className="text-xs"
                        >
                          #{channel}
                        </Badge>
                      ))}
                      {config.connectedChannels.length > 8 && (
                        <Badge variant="outline" className="text-xs">
                          +{config.connectedChannels.length - 8} more
                        </Badge>
                      )}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        );

      case "github":
        return (
          <div className="space-y-4">
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-3">
                Installation Details
              </h4>
              <div className="bg-gray-50 rounded-lg p-4 space-y-3">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  {config.installationId && (
                    <div>
                      <span className="text-gray-500">Installation ID</span>
                      <p className="font-mono text-xs">
                        {config.installationId}
                      </p>
                    </div>
                  )}
                  <div>
                    <span className="text-gray-500">Webhook Status</span>
                    <p className="font-medium text-green-600">Active</p>
                  </div>
                </div>
              </div>
            </div>

            {config.connectedRepos && (
              <div>
                <h4 className="text-sm font-medium text-gray-900 mb-3">
                  Repository Access
                </h4>
                <div className="bg-gray-50 rounded-lg p-4">
                  <div className="text-sm mb-2">
                    <span className="text-gray-500">
                      Connected Repositories:{" "}
                    </span>
                    <span className="font-medium">
                      {config.connectedRepos.length}
                    </span>
                  </div>
                  {config.connectedRepos.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {config.connectedRepos.slice(0, 6).map((repo) => (
                        <Badge key={repo} variant="outline" className="text-xs">
                          {repo}
                        </Badge>
                      ))}
                      {config.connectedRepos.length > 6 && (
                        <Badge variant="outline" className="text-xs">
                          +{config.connectedRepos.length - 6} more
                        </Badge>
                      )}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        );

      case "aws":
        return (
          <div className="space-y-4">
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-3">
                AWS Configuration
              </h4>
              <div className="bg-gray-50 rounded-lg p-4 space-y-3">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  {config.region && (
                    <div>
                      <span className="text-gray-500">Default Region</span>
                      <p className="font-medium">{config.region}</p>
                    </div>
                  )}
                  {config.accountId && (
                    <div>
                      <span className="text-gray-500">Account ID</span>
                      <p className="font-mono text-xs">{config.accountId}</p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        );

      case "gcp":
        return (
          <div className="space-y-4">
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-3">
                Google Cloud Configuration
              </h4>
              <div className="bg-gray-50 rounded-lg p-4 space-y-3">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  {config.projectId && (
                    <div>
                      <span className="text-gray-500">Project ID</span>
                      <p className="font-mono text-xs">{config.projectId}</p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        );

      default:
        return (
          <div>
            <h4 className="text-sm font-medium text-gray-900 mb-3">
              Configuration
            </h4>
            <div className="bg-gray-50 rounded-lg p-4">
              <p className="text-sm text-gray-600">
                Configuration details will be displayed here once available.
              </p>
            </div>
          </div>
        );
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center space-x-2">
          <Settings className="h-5 w-5" />
          <span>Configuration</span>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <div>
          <h4 className="text-sm font-medium text-gray-900 mb-3">
            Authentication
          </h4>
          {renderAuthTypeInfo()}
        </div>

        {renderCapabilities()}

        {renderConnectorSpecificSettings()}

        <div className="border-t pt-4">
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
            <div className="flex items-start space-x-2">
              <Shield className="h-4 w-4 text-blue-600 mt-0.5" />
              <div className="text-sm">
                <p className="text-blue-800 font-medium">
                  Security Information
                </p>
                <p className="text-blue-700 mt-1">
                  All credentials are encrypted using AES-256-GCM and stored
                  securely. InfraLayer only accesses the permissions explicitly
                  granted during setup.
                </p>
              </div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};
