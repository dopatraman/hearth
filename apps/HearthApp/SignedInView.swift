import SwiftUI

/// Deliberately minimal placeholder for Slice 2. The real backlog + personal
/// list lands in Slice 3 and replaces this view entirely.
struct SignedInView: View {
    let user: HearthUser
    @Environment(AuthStore.self) private var auth

    var body: some View {
        VStack(spacing: 24) {
            Text("Hey \(user.displayName ?? "there")")
                .font(.system(.largeTitle, design: .rounded).weight(.semibold))
                .foregroundStyle(Color.hearthTextPrimary)

            Text("Backlog lands in the next slice.")
                .font(.subheadline)
                .foregroundStyle(Color.hearthTextSecondary)

            Button("Sign out") {
                auth.signOut()
            }
            .padding(.top, 32)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color.hearthBackground.ignoresSafeArea())
    }
}

#Preview {
    SignedInView(user: HearthUser(id: "preview", displayName: "Alice", notificationsEnabled: true))
        .environment(AuthStore())
}
