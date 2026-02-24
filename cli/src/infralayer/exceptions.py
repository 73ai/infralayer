class CLIError(Exception):
    pass


class AuthValidationError(CLIError):
    pass


class TokenRefreshError(CLIError):
    pass


class GCPCredentialError(CLIError):
    pass


class GKEClusterError(CLIError):
    pass


class ContainerSetupError(CLIError):
    pass
