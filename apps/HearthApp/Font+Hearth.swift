import SwiftUI

/// Hearth typography — one display face (Bricolage Grotesque) and one body
/// face (Geist). Falls back to the system font if the custom font failed to
/// register (e.g., preview on a machine that hasn't bundled the TTF).
extension Font {
    /// Bricolage Grotesque — the warm display face used for the Backlog title
    /// and section-level labels that want a touch of character.
    static func hearthDisplay(size: CGFloat, weight: Weight = .semibold) -> Font {
        .custom("BricolageGrotesque", size: size).weight(weight)
    }

    /// Geist — the body face. Clean, modern, calm.
    static func hearthBody(size: CGFloat, weight: Weight = .regular) -> Font {
        let family: String
        switch weight {
        case .medium: family = "Geist-Medium"
        case .semibold, .bold, .heavy, .black: family = "Geist-SemiBold"
        default: family = "Geist-Regular"
        }
        return .custom(family, size: size)
    }
}
