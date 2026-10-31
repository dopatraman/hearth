import SwiftUI

/// Root tab view shown when the user is signed in. Two tabs: the Backlog
/// (open chores + add + claim + complete) and the Graveyard (completed +
/// search).
struct RootTabView: View {
    @State private var backlog: BacklogStore
    @State private var graveyard: GraveyardStore
    @State private var members: MembersStore
    @State private var selection: Tab = .backlog
    @Environment(\.scenePhase) private var scenePhase

    enum Tab: Hashable { case backlog, graveyard }

    init(sessionProvider: @escaping () -> String?) {
        _backlog = State(initialValue: BacklogStore(session: sessionProvider))
        _graveyard = State(initialValue: GraveyardStore(session: sessionProvider))
        _members = State(initialValue: MembersStore(session: sessionProvider))
    }

    var body: some View {
        TabView(selection: $selection) {
            BacklogView(initialStore: backlog, members: members)
                .tabItem {
                    Label("Backlog", systemImage: "tray.full")
                }
                .tag(Tab.backlog)

            GraveyardView(initialStore: graveyard, members: members)
                .tabItem {
                    Label("Done", systemImage: "checkmark.circle")
                }
                .tag(Tab.graveyard)
        }
        .tint(Color.hearthLavenderInk)
        .tabBarMinimizeBehavior(.onScrollDown)
        .onChange(of: selection) { _, newTab in
            // Reload graveyard when the user taps into it so it's fresh.
            if newTab == .graveyard {
                Task { await graveyard.reload() }
            }
        }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active {
                Task { await refresh() }
            }
        }
        .onAppear {
            PushRegistration.shared.onRefresh = { await refresh() }
        }
    }

    private func refresh() async {
        await backlog.reload()
        await graveyard.reload()
        await members.load()
    }
}
