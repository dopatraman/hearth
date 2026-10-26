import SwiftUI

/// Hearth palette — semantic names, light + dark values from the build plan.
/// A color here always carries meaning: a lavender highlight ALWAYS means
/// "mine"; a peach badge ALWAYS means "overdue". Users learn the system by
/// using it.
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

    /// Primary / "mine" — a chore claimed by the current user.
    static let hearthLavender = Color(
        light: Color(red: 0xC9 / 255, green: 0xB6 / 255, blue: 0xE4 / 255),
        dark: Color(red: 0x8A / 255, green: 0x7B / 255, blue: 0xA8 / 255)
    )

    /// Success / completed — the mint check button and graveyard accents.
    static let hearthMint = Color(
        light: Color(red: 0xB6 / 255, green: 0xE4 / 255, blue: 0xCF / 255),
        dark: Color(red: 0x7A / 255, green: 0xA8 / 255, blue: 0x96 / 255)
    )

    /// Warning / overdue — chores past their will_complete_by.
    static let hearthPeach = Color(
        light: Color(red: 0xF4 / 255, green: 0xC4 / 255, blue: 0xA3 / 255),
        dark: Color(red: 0xB8 / 255, green: 0x8A / 255, blue: 0x6E / 255)
    )

    /// Info / claimed-by-other — avatar chip for chores someone else owns.
    static let hearthSky = Color(
        light: Color(red: 0xA3 / 255, green: 0xC9 / 255, blue: 0xF4 / 255),
        dark: Color(red: 0x6B / 255, green: 0x94 / 255, blue: 0xC2 / 255)
    )

    /// Ink color for text/initial drawn on `hearthLavender` surfaces (self avatar).
    static let hearthLavenderInk = Color(
        light: Color(red: 0x4B / 255, green: 0x3A / 255, blue: 0x8C / 255),
        dark: Color(red: 0xC7 / 255, green: 0xB9 / 255, blue: 0xEC / 255)
    )

    /// Ink color for text/initial drawn on `hearthSky` surfaces (other-member avatar).
    static let hearthSkyInk = Color(
        light: Color(red: 0x1F / 255, green: 0x5A / 255, blue: 0x8F / 255),
        dark: Color(red: 0xB4 / 255, green: 0xD1 / 255, blue: 0xEF / 255)
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
