import SwiftUI

/// Root switcher. Shows the sign-in screen until AuthStore reports a signed-in
/// user; otherwise mounts the backlog with a BacklogStore wired to the current
/// session token.
///
/// Also handles the "a universal invite link opened the app" flow — when
/// AuthStore has a pendingInviteCode AND the user is signed in, show a
/// confirmation alert.
struct ContentView: View {
    @Environment(AuthStore.self) private var auth

    var body: some View {
        ZStack {
            switch auth.phase {
            case .restoring:
                Color.hearthBackground.ignoresSafeArea()
                ProgressView()
            case .signedOut, .signingIn:
                SignInScreen()
            case .signedIn:
                RootTabView(sessionProvider: { Keychain().read() })
                    .id(auth.membershipVersion)
            }
        }
        .animation(.easeInOut(duration: 0.3), value: auth.phase)
        .task {
            if case .restoring = auth.phase {
                await auth.bootstrap()
            }
        }
        .alert(
            "Join household?",
            isPresented: Binding(
                get: { auth.pendingInviteCode != nil && isSignedIn },
                set: { if !$0 { auth.dismissPendingInvite() } }
            ),
            presenting: auth.pendingInviteCode
        ) { code in
            Button("Accept") {
                Task { await auth.acceptInvite(code: code) }
            }
            Button("Not now", role: .cancel) {
                auth.dismissPendingInvite()
            }
        } message: { code in
            Text("You've been invited with code \(formatted(code)). Accept to see this household's chores.")
        }
    }

    private var isSignedIn: Bool {
        if case .signedIn = auth.phase { return true }
        return false
    }

    private func formatted(_ code: String) -> String {
        guard code.count == 6 else { return code }
        let i = code.index(code.startIndex, offsetBy: 3)
        return code[..<i] + "-" + code[i...]
    }
}

#Preview {
    ContentView()
        .environment(AuthStore())
}
