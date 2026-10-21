import SwiftUI

/// Root switcher. Shows the sign-in screen until AuthStore reports a signed-in
/// user; the signed-in placeholder (soon: backlog) once it does.
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
            case .signedIn(let user):
                SignedInView(user: user)
            }
        }
        .animation(.easeInOut(duration: 0.3), value: auth.phase)
        .task {
            if case .restoring = auth.phase {
                await auth.bootstrap()
            }
        }
    }
}

#Preview {
    ContentView()
        .environment(AuthStore())
}
