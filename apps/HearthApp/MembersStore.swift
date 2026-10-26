import Foundation

struct HearthMember: Codable, Equatable, Identifiable, Sendable {
    let id: String
    let displayName: String?

    enum CodingKeys: String, CodingKey {
        case id
        case displayName = "display_name"
    }
}

struct MembersResponse: Codable, Sendable {
    let members: [HearthMember]
}

@Observable
@MainActor
final class MembersStore {
    private(set) var members: [HearthMember] = []

    private let api: HearthAPI
    private let session: () -> String?

    init(api: HearthAPI = HearthAPI(), session: @escaping () -> String?) {
        self.api = api
        self.session = session
    }

    func load() async {
        guard let token = session() else { return }
        do {
            members = try await api.members(sessionToken: token)
        } catch {
            // Silent — avatars gracefully degrade to initials unresolved.
        }
    }

    func displayName(for userID: String) -> String? {
        Self.displayName(for: userID, in: members)
    }

    func initial(for userID: String) -> String? {
        Self.initial(for: userID, in: members)
    }

    nonisolated static func displayName(for userID: String, in members: [HearthMember]) -> String? {
        members.first(where: { $0.id == userID })?.displayName
    }

    nonisolated static func initial(for userID: String, in members: [HearthMember]) -> String? {
        guard let member = members.first(where: { $0.id == userID }) else { return nil }
        guard let name = member.displayName else { return "?" }
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let firstChar = trimmed.first else { return "?" }
        return String(firstChar).uppercased()
    }
}
