export class ApiError extends Error {
  constructor(message: string, readonly requestId?: string, readonly status?: number) {
    super(message);
    this.name = new.target.name;
  }
}

export class ConfigurationError extends ApiError {}
export class CancelledError extends ApiError {}
export class TimeoutError extends ApiError {}
export class NetworkError extends ApiError {}
export class InvalidResponseError extends ApiError {}
export class InvalidRequestError extends ApiError {}
export class UnauthorizedError extends ApiError {}
export class ForbiddenError extends ApiError {}
export class EndpointNotFoundError extends ApiError {}
export class PayloadTooLargeError extends ApiError {}
export class RateLimitError extends ApiError {}
export class BackendUnavailableError extends ApiError {}
