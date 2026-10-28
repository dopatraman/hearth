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

// MARK: - Chores

struct HearthChore: Codable, Equatable, Identifiable {
    let id: String
    let listId: String
    let creatorId: String
    let title: String
    let notes: String?
    let state: String
    let claimedBy: String?
    let claimedAt: String?
    let willCompleteBy: String?
    let completedBy: String?
    let completedAt: String?
    let createdAt: String
    let updatedAt: String
    let overdue: Bool
    let snoozedUntil: String?

    init(
        id: String,
        listId: String,
        creatorId: String,
        title: String,
        notes: String?,
        state: String,
        claimedBy: String?,
        claimedAt: String?,
        willCompleteBy: String?,
        completedBy: String?,
        completedAt: String?,
        createdAt: String,
        updatedAt: String,
        overdue: Bool,
        snoozedUntil: String? = nil
    ) {
        self.id = id
        self.listId = listId
        self.creatorId = creatorId
        self.title = title
        self.notes = notes
        self.state = state
        self.claimedBy = claimedBy
        self.claimedAt = claimedAt
        self.willCompleteBy = willCompleteBy
        self.completedBy = completedBy
        self.completedAt = completedAt
        self.createdAt = createdAt
        self.updatedAt = updatedAt
        self.overdue = overdue
        self.snoozedUntil = snoozedUntil
    }

    enum CodingKeys: String, CodingKey {
        case id
        case listId = "list_id"
        case creatorId = "creator_id"
        case title
        case notes
        case state
        case claimedBy = "claimed_by"
        case claimedAt = "claimed_at"
        case willCompleteBy = "will_complete_by"
        case completedBy = "completed_by"
        case completedAt = "completed_at"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case overdue
        case snoozedUntil = "snoozed_until"
    }
}

extension HearthChore {
    func with(snoozedUntil: String?) -> HearthChore {
        HearthChore(
            id: id,
            listId: listId,
            creatorId: creatorId,
            title: title,
            notes: notes,
            state: state,
            claimedBy: claimedBy,
            claimedAt: claimedAt,
            willCompleteBy: willCompleteBy,
            completedBy: completedBy,
            completedAt: completedAt,
            createdAt: createdAt,
            updatedAt: updatedAt,
            overdue: overdue,
            snoozedUntil: snoozedUntil
        )
    }
}

struct CreateChoreRequest: Codable {
    let title: String
    let notes: String?
    let claim: Bool
    let willCompleteBy: String?

    enum CodingKeys: String, CodingKey {
        case title
        case notes
        case claim
        case willCompleteBy = "will_complete_by"
    }
}

struct BacklogResponse: Codable, Equatable {
    let chores: [HearthChore]
}

// MARK: - Invites

struct InviteDTO: Codable, Equatable {
    let code: String
    let listId: String
    let expiresAt: String

    enum CodingKeys: String, CodingKey {
        case code
        case listId = "list_id"
        case expiresAt = "expires_at"
    }
}

struct AcceptInviteResponse: Codable, Equatable {
    let listId: String

    enum CodingKeys: String, CodingKey {
        case listId = "list_id"
    }
}
