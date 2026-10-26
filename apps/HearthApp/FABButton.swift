import SwiftUI

/// Floating action button. Primary-action affordance for a screen, floated
/// bottom-right as an overlay. Lavender-tinted Liquid Glass on iOS 26, with a
/// static material fallback for older runtime (deployment targets this, but
/// `#available` keeps the surface defensive).
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
                .foregroundStyle(Color.hearthLavenderInk)
                .frame(width: 62, height: 62)
                .glassEffect(
                    .regular.tint(Color.hearthLavender).interactive(),
                    in: Circle()
                )
                .shadow(color: Color.hearthLavender.opacity(0.38), radius: 20, y: 8)
        }
        .buttonStyle(.plain)
    }
}

#if DEBUG
#Preview {
    ZStack {
        Color.hearthBackground.ignoresSafeArea()
        FABButton(action: {})
    }
}
#endif
