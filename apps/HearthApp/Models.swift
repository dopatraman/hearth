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

// MARK: - Items

struct HearthItem: Codable, Equatable, Identifiable {
    let id: String
    let listId: String
    let creatorId: String
    let title: String
    let notes: String?
    let state: String
    let claimedBy: String?
    let claimedAt: String?
    let assignedBy: String?
    let willCompleteBy: String?
    let completedBy: String?
    let completedAt: String?
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case listId = "list_id"
        case creatorId = "creator_id"
        case title
        case notes
        case state
        case claimedBy = "claimed_by"
        case claimedAt = "claimed_at"
        case assignedBy = "assigned_by"
        case willCompleteBy = "will_complete_by"
        case completedBy = "completed_by"
        case completedAt = "completed_at"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct CreateItemRequest: Codable {
    let title: String
    let notes: String?
}

struct BacklogResponse: Codable, Equatable {
    let items: [HearthItem]
}
