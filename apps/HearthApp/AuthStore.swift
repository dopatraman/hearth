import AuthenticationServices
import Foundation

/// AuthStore owns the authenticated-user state and the lifecycle calls that
/// mutate it (sign in, sign out, restore from Keychain). SwiftUI reads it via
/// @Environment.
@Observable
@MainActor
final class AuthStore {
    enum Phase: Equatable {
        case restoring       // launching, checking Keychain
        case signedOut
        case signedIn(HearthUser)
        case signingIn       // Apple sheet shown or network in flight
    }

    private(set) var phase: Phase = .restoring
    private(set) var lastError: String?

    /// Set when a universal link like /invite/<code> opens the app. Consumed
    /// by ContentView, which prompts the user to accept and clears via
    /// acceptPendingInvite() / dismissPendingInvite().
    var pendingInviteCode: String?

    private let api: HearthAPI
    private let keychain: Keychain

    init(api: HearthAPI = HearthAPI(), keychain: Keychain = Keychain()) {
        self.api = api
        self.keychain = keychain
    }

    /// Called from HearthApp.onAppear or the top-level task to restore the
    /// previous session if one exists.
    func bootstrap() async {
        guard let token = keychain.read() else {
            phase = .signedOut
            return
        }
        do {
            let user = try await api.me(sessionToken: token)
            phase = .signedIn(user)
        } catch {
            // Token invalid / server unreachable — treat as signed out.
            try? keychain.delete()
            phase = .signedOut
        }
    }

    /// Handles the result of a SignInWithAppleButton completion.
    func handleAppleSignIn(_ result: Result<ASAuthorization, Error>) async {
        lastError = nil
        phase = .signingIn

        switch result {
        case .failure(let error):
            // User cancelled is noisy to show; just revert.
            if (error as NSError).code == ASAuthorizationError.canceled.rawValue {
                phase = .signedOut
                return
            }
            lastError = error.localizedDescription
            phase = .signedOut

        case .success(let auth):
            guard let credential = auth.credential as? ASAuthorizationAppleIDCredential,
                  let identityTokenData = credential.identityToken,
                  let identityToken = String(data: identityTokenData, encoding: .utf8)
            else {
                lastError = "Apple did not return an identity token."
                phase = .signedOut
                return
            }

            let displayName: String? = {
                let parts = [credential.fullName?.givenName, credential.fullName?.familyName]
                    .compactMap { $0 }
                    .filter { !$0.isEmpty }
                return parts.isEmpty ? nil : parts.joined(separator: " ")
            }()

            do {
                let response = try await api.signInWithApple(
                    idToken: identityToken,
                    displayName: displayName
                )
                try keychain.save(response.sessionToken)
                phase = .signedIn(response.user)
            } catch {
                lastError = error.localizedDescription
                phase = .signedOut
            }
        }
    }

    func signOut() {
        try? keychain.delete()
        phase = .signedOut
    }

    /// Accepts the stashed invite code. Returns true on success. Clears
    /// pendingInviteCode regardless of outcome so the alert dismisses.
    @discardableResult
    func acceptPendingInvite() async -> Bool {
        guard let code = pendingInviteCode, let token = keychain.read() else {
            pendingInviteCode = nil
            return false
        }
        pendingInviteCode = nil
        do {
            _ = try await api.acceptInvite(sessionToken: token, code: code)
            return true
        } catch {
            lastError = error.localizedDescription
            return false
        }
    }

    func dismissPendingInvite() {
        pendingInviteCode = nil
    }

    #if DEBUG
    convenience init(previewSignedInUser user: HearthUser) {
        self.init()
        self.phase = .signedIn(user)
    }
    #endif
}
