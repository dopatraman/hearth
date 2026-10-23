import SwiftUI

/// Generates and shows a 6-digit invite code the user can share with a
/// household member. Share button produces a universal link that, when tapped
/// on an iPhone with Hearth installed, opens the app and prompts to accept.
struct InviteSheet: View {
    @Environment(\.dismiss) private var dismiss

    @State private var invite: InviteDTO?
    @State private var error: String?
    @State private var loading: Bool = false

    var body: some View {
        NavigationStack {
            ZStack {
                Color.hearthBackground.ignoresSafeArea()

                VStack(spacing: 32) {
                    Spacer()

                    if let invite {
                        Text("Share this code")
                            .font(.title3)
                            .foregroundStyle(Color.hearthTextSecondary)

                        Text(formatted(code: invite.code))
                            .font(.system(size: 56, weight: .semibold, design: .rounded))
                            .foregroundStyle(Color.hearthTextPrimary)
                            .monospacedDigit()
                            .padding(.horizontal, 16)
                            .padding(.vertical, 12)
                            .background(
                                RoundedRectangle(cornerRadius: 16)
                                    .fill(.ultraThinMaterial)
                            )

                        Text("Valid for 24 hours.")
                            .font(.footnote)
                            .foregroundStyle(Color.hearthTextSecondary)

                        ShareLink(item: shareURL(for: invite.code)) {
                            Label("Share link", systemImage: "square.and.arrow.up")
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 10)
                        }
                        .buttonStyle(.borderedProminent)
                        .padding(.horizontal, 40)
                    } else if let error {
                        Image(systemName: "exclamationmark.triangle")
                            .font(.largeTitle)
                            .foregroundStyle(.red)
                        Text(error)
                            .font(.footnote)
                            .foregroundStyle(Color.hearthTextSecondary)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal, 32)
                        Button("Retry") { Task { await load() } }
                            .buttonStyle(.borderedProminent)
                    } else {
                        ProgressView()
                            .controlSize(.large)
                    }

                    Spacer()
                }
            }
            .navigationTitle("Invite")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close") { dismiss() }
                }
            }
            .task { await load() }
        }
        .presentationDetents([.medium, .large])
    }

    private func load() async {
        loading = true
        defer { loading = false }
        error = nil
        guard let token = Keychain().read() else {
            error = "Not signed in."
            return
        }
        do {
            invite = try await HearthAPI().createInvite(sessionToken: token)
        } catch {
            self.error = error.localizedDescription
        }
    }

    /// Formats 123456 as 123-456 for readability.
    private func formatted(code: String) -> String {
        guard code.count == 6 else { return code }
        let i = code.index(code.startIndex, offsetBy: 3)
        return code[..<i] + "-" + code[i...]
    }

    private func shareURL(for code: String) -> URL {
        URL(string: "https://api.hearth.prakashvenkat.com/invite/\(code)")!
    }
}

#Preview {
    InviteSheet()
}
