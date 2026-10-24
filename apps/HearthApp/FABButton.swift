import SwiftUI

/// Floating action button. Primary-action affordance for a screen, floated
/// bottom-right as an overlay. Styled with ultra-thin material + a subtle
/// shadow; swap to Liquid Glass's `.glassEffect` in the design-polish pass
/// once we're on an Xcode toolchain that has the modifier.
struct FABButton: View {
    let systemImage: String
    let action: () -> Void

    init(systemImage: String = "plus", action: @escaping () -> Void) {
        self.systemImage = systemImage
        self.action = action
    }

    var body: some View {
        Button(action: action) {
            Image(systemName: systemImage)
                .font(.system(size: 24, weight: .semibold))
                .foregroundStyle(Color.hearthTextPrimary)
                .frame(width: 60, height: 60)
                .background(.ultraThinMaterial, in: Circle())
                .overlay(Circle().stroke(Color.hearthTextSecondary.opacity(0.2), lineWidth: 0.5))
                .shadow(color: Color.black.opacity(0.1), radius: 10, y: 4)
        }
        .buttonStyle(.plain)
    }
}

#Preview {
    ZStack {
        Color.hearthBackground.ignoresSafeArea()
        FABButton(action: {})
    }
}
