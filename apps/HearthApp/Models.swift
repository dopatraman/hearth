import Foundation

// Hand-maintained types that mirror the Go server's JSON contract.
// Grows as the API surface expands slice by slice.

struct HealthResponse: Codable, Equatable {
    let status: String
}

struct AuthAppleRequest: Codable {
    let idToken: String
    let displayName: String?

    enum CodingKeys: String, CodingKey {
        case idToken = "id_token"
        case displayName = "display_name"
    }
}

struct AuthAppleResponse: Codable, Equatable {
    let sessionToken: String
    let user: HearthUser

    enum CodingKeys: String, CodingKey {
        case sessionToken = "session_token"
        case user
    }
}

struct HearthUser: Codable, Equatable, Identifiable {
    let id: String
    let displayName: String?
    let notificationsEnabled: Bool

    enum CodingKeys: String, CodingKey {
        case id
        case displayName = "display_name"
        case notificationsEnabled = "notifications_enabled"
    }
}

struct APIErrorResponse: Codable, Equatable {
    let error: String
}
