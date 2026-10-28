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
