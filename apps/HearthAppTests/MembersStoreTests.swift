import Foundation
import Testing
@testable import HearthApp

@Suite("MembersStore")
struct MembersStoreTests {
    @Test("resolves display name after load")
    func resolvesDisplayNameAfterLoad() {
        let members = [
            HearthMember(id: "u_alice", displayName: "Alice"),
            HearthMember(id: "u_maya", displayName: "Maya"),
        ]
        #expect(MembersStore.displayName(for: "u_maya", in: members) == "Maya")
        #expect(MembersStore.displayName(for: "u_alice", in: members) == "Alice")
    }

    @Test("initial falls back to ? for empty display name")
    func initialFallsBackToQuestionMarkForEmptyName() {
        let members = [HearthMember(id: "u_blank", displayName: "")]
        #expect(MembersStore.initial(for: "u_blank", in: members) == "?")
    }

    @Test("initial falls back to ? for nil display name")
    func initialFallsBackToQuestionMarkForNilName() {
        let members = [HearthMember(id: "u_nil", displayName: nil)]
        #expect(MembersStore.initial(for: "u_nil", in: members) == "?")
    }

    @Test("initial returns uppercased first character")
    func initialReturnsUppercasedFirstCharacter() {
        let members = [
            HearthMember(id: "u_maya", displayName: "maya"),
            HearthMember(id: "u_alice", displayName: "Alice"),
        ]
        #expect(MembersStore.initial(for: "u_maya", in: members) == "M")
        #expect(MembersStore.initial(for: "u_alice", in: members) == "A")
    }

    @Test("displayName returns nil for unknown id")
    func displayNameReturnsNilForUnknownID() {
        let members = [HearthMember(id: "u_alice", displayName: "Alice")]
        #expect(MembersStore.displayName(for: "u_ghost", in: members) == nil)
    }

    @Test("initial returns nil for unknown id")
    func initialReturnsNilForUnknownID() {
        let members = [HearthMember(id: "u_alice", displayName: "Alice")]
        #expect(MembersStore.initial(for: "u_ghost", in: members) == nil)
    }
}
