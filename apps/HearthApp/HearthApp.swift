import SwiftUI

@main
struct HearthApp: App {
    @State private var auth = AuthStore()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(auth)
                .onOpenURL { url in
                    if let code = inviteCode(from: url) {
                        auth.pendingInviteCode = code
                    }
                }
        }
    }

    /// Extracts the code from a /invite/<CODE> universal link.
    /// Non-matching URLs return nil.
    private func inviteCode(from url: URL) -> String? {
        let path = url.path  // e.g., "/invite/123456"
        let parts = path.split(separator: "/", omittingEmptySubsequences: true)
        guard parts.count == 2, parts[0] == "invite" else { return nil }
        let code = String(parts[1])
        return code.isEmpty ? nil : code
    }
}
