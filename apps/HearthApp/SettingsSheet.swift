import SwiftUI

/// Account + household settings. Minimal for Slice 4 — invite + sign out.
/// Grows with notification toggle (Slice 8) etc.
struct SettingsSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(AuthStore.self) private var auth
    @State private var presentingInvite: Bool = false
    @State private var notificationsOn: Bool = true
    @State private var notificationsSyncing: Bool = false
    private let api = HearthAPI()

    private var currentUser: HearthUser? {
        if case .signedIn(let user) = auth.phase { return user }
        return nil
    }

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

                Section("Notifications") {
                    Toggle("Push notifications", isOn: $notificationsOn)
                        .disabled(notificationsSyncing)
                        .onChange(of: notificationsOn) { _, newValue in
                            Task { await setNotifications(newValue) }
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
            .onAppear {
                if let u = currentUser { notificationsOn = u.notificationsEnabled }
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

    private func setNotifications(_ enabled: Bool) async {
        guard let token = Keychain().read() else { return }
        notificationsSyncing = true
        defer { notificationsSyncing = false }
        do {
            try await api.setNotificationsEnabled(sessionToken: token, enabled: enabled)
            if enabled {
                await PushRegistration.shared.requestPermissionAndRegister()
            }
        } catch {
            notificationsOn = !enabled
        }
    }
}

#Preview {
    SettingsSheet()
        .environment(AuthStore())
}
