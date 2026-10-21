import AuthenticationServices
import SwiftUI

struct SignInScreen: View {
    @Environment(AuthStore.self) private var auth
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        VStack {
            Spacer(minLength: 0)

            VStack(spacing: 12) {
                Text("Hearth")
                    .font(.system(size: 72, weight: .semibold, design: .rounded))
                    .foregroundStyle(Color.hearthTextPrimary)
                Text("Shared chores for your household")
                    .font(.title3)
                    .foregroundStyle(Color.hearthTextSecondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, 32)
            }

            Spacer(minLength: 0)

            VStack(spacing: 16) {
                if let error = auth.lastError {
                    Text(error)
                        .font(.footnote)
                        .foregroundStyle(.red)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 32)
                }

                SignInWithAppleButton(.signIn) { request in
                    request.requestedScopes = [.fullName, .email]
                } onCompletion: { result in
                    Task { await auth.handleAppleSignIn(result) }
                }
                .signInWithAppleButtonStyle(colorScheme == .dark ? .white : .black)
                .frame(height: 54)
                .padding(.horizontal, 40)
            }
            .padding(.bottom, 48)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color.hearthBackground.ignoresSafeArea())
    }
}

#Preview {
    SignInScreen()
        .environment(AuthStore())
}
