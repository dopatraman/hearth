import SwiftUI

/// Hearth palette — semantic names, light + dark values from the build plan.
/// Expand incrementally; only what Slice 2 needs is defined for now.
extension Color {
    /// Warm cream in light, near-black warm in dark.
    static let hearthBackground = Color(
        light: Color(red: 0xFB / 255, green: 0xF8 / 255, blue: 0xF3 / 255),
        dark: Color(red: 0x1C / 255, green: 0x1A / 255, blue: 0x17 / 255)
    )

    /// Warm, not pure. Primary text.
    static let hearthTextPrimary = Color(
        light: Color(red: 0x2A / 255, green: 0x26 / 255, blue: 0x21 / 255),
        dark: Color(red: 0xF0 / 255, green: 0xEC / 255, blue: 0xE6 / 255)
    )

    /// Metadata, timestamps, hints.
    static let hearthTextSecondary = Color(
        light: Color(red: 0x6B / 255, green: 0x65 / 255, blue: 0x60 / 255),
        dark: Color(red: 0xA6 / 255, green: 0xA0 / 255, blue: 0x9A / 255)
    )
}

private extension Color {
    init(light: Color, dark: Color) {
        self.init(uiColor: UIColor { trait in
            trait.userInterfaceStyle == .dark
                ? UIColor(dark)
                : UIColor(light)
        })
    }
}
