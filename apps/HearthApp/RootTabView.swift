import SwiftUI

/// Root tab view shown when the user is signed in. Two tabs: the Backlog
/// (open chores + add + claim + complete) and the Graveyard (completed +
/// search).
struct RootTabView: View {
    @State private var backlog: BacklogStore
    @State private var graveyard: GraveyardStore
    @State private var members: MembersStore
    @State private var selection: TabID = .backlog

    enum TabID: Hashable { case backlog, graveyard }

    init(sessionProvider: @escaping () -> String?) {
        _backlog = State(initialValue: BacklogStore(session: sessionProvider))
        _graveyard = State(initialValue: GraveyardStore(session: sessionProvider))
        _members = State(initialValue: MembersStore(session: sessionProvider))
    }

    var body: some View {
        TabView(selection: $selection) {
            Tab("Backlog", systemImage: "tray.full", value: TabID.backlog) {
                BacklogView(initialStore: backlog, members: members)
                    .toolbar(.visible, for: .tabBar)
                    .toolbarBackground(.visible, for: .tabBar)
            }

            Tab("Done", systemImage: "checkmark.circle", value: TabID.graveyard) {
                GraveyardView(initialStore: graveyard)
                    .toolbar(.visible, for: .tabBar)
                    .toolbarBackground(.visible, for: .tabBar)
            }
        }
        .tint(Color.hearthLavenderInk)
        .onChange(of: selection) { _, newTab in
            if newTab == .graveyard {
                Task { await graveyard.reload() }
            }
        }
    }
}

#if DEBUG
#Preview("Signed in — populated backlog") {
    let selfUser = HearthUser(id: "u_self", displayName: "Venkat", notificationsEnabled: true)
    let members = [
        HearthMember(id: "u_self", displayName: "Venkat"),
        HearthMember(id: "u_maya", displayName: "Maya"),
    ]
    let chores: [HearthChore] = [
        HearthChore(
            id: "c1", listId: "l", creatorId: "u_self",
            title: "Clean out the fridge", notes: "Toss anything past its date.",
            state: "open",
            claimedBy: "u_self", claimedAt: "2026-10-23T14:00:00.000Z",
            willCompleteBy: "2026-10-23T23:59:59.000Z",
            completedBy: nil, completedAt: nil,
            createdAt: "2026-10-20T10:00:00.000Z",
            updatedAt: "2026-10-20T10:00:00.000Z",
            overdue: true
        ),
        HearthChore(
            id: "c2", listId: "l", creatorId: "u_self",
            title: "Water the plants", notes: nil, state: "open",
            claimedBy: "u_self", claimedAt: "2026-10-23T14:00:00.000Z",
            willCompleteBy: nil, completedBy: nil, completedAt: nil,
            createdAt: "2026-10-21T10:00:00.000Z",
            updatedAt: "2026-10-21T10:00:00.000Z",
            overdue: false
        ),
        HearthChore(
            id: "c3", listId: "l", creatorId: "u_self",
            title: "Wipe down the stove", notes: nil, state: "open",
            claimedBy: nil, claimedAt: nil, willCompleteBy: nil,
            completedBy: nil, completedAt: nil,
            createdAt: "2026-10-22T10:00:00.000Z",
            updatedAt: "2026-10-22T10:00:00.000Z",
            overdue: false
        ),
    ]
    let auth = AuthStore(previewSignedInUser: selfUser)
    let backlog = BacklogStore(preview: chores)
    let memberStore = MembersStore(preview: members)
    return RootTabView(sessionProvider: { "preview-token" })
        .environment(auth)
        .onAppear {
            _ = backlog
            _ = memberStore
        }
}
#endif
