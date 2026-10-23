import SwiftUI

/// Account + household settings. Minimal for Slice 4 — invite + sign out.
/// Grows with notification toggle (Slice 8) etc.
struct SettingsSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(AuthStore.self) private var auth
    @State private var presentingInvite: Bool = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Household") {
                    Button {
                        presentingInvite = true
                    } label: {
                        Label("Invite someone", systemImage: "person.badge.plus")
                            .foregroundStyle(Color.hearthTextPrimary)
                    }
                }

                Section {
                    Button(role: .destructive) {
                        auth.signOut()
                        dismiss()
                    } label: {
                        Text("Sign out")
                    }
                }
            }
            .scrollContentBackground(.hidden)
            .background(Color.hearthBackground)
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { dismiss() }
                }
            }
            .sheet(isPresented: $presentingInvite) {
                InviteSheet()
            }
        }
        .presentationDetents([.medium, .large])
    }
}

#Preview {
    SettingsSheet()
        .environment(AuthStore())
}
