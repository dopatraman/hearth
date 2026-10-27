import Foundation

enum WhenishBucket: String, CaseIterable, Identifiable {
    case tonight
    case tomorrow
    case weekend
    case someday

    var id: String { rawValue }

    var label: String {
        switch self {
        case .tonight: return "Tonight"
        case .tomorrow: return "Tomorrow"
        case .weekend: return "This weekend"
        case .someday: return "No rush"
        }
    }

    var toastSuffix: String {
        switch self {
        case .tonight: return " by tonight"
        case .tomorrow: return " by tomorrow"
        case .weekend: return " by the weekend"
        case .someday: return ""
        }
    }

    func resolve(now: Date, calendar: Calendar = .current) -> Date? {
        switch self {
        case .someday:
            return nil
        case .tonight:
            return endOfDay(for: now, calendar: calendar)
        case .tomorrow:
            guard let tomorrow = calendar.date(byAdding: .day, value: 1, to: now) else { return nil }
            return endOfDay(for: tomorrow, calendar: calendar)
        case .weekend:
            return endOfDay(for: nextOrSameSunday(from: now, calendar: calendar), calendar: calendar)
        }
    }

    private func endOfDay(for date: Date, calendar: Calendar) -> Date {
        let start = calendar.startOfDay(for: date)
        return start.addingTimeInterval(24 * 60 * 60 - 0.001)
    }

    private func nextOrSameSunday(from date: Date, calendar: Calendar) -> Date {
        let weekday = calendar.component(.weekday, from: date)
        let daysUntilSunday = (8 - weekday) % 7
        return calendar.date(byAdding: .day, value: daysUntilSunday, to: date) ?? date
    }
}

extension WhenishBucket {
    static func iso8601(from date: Date) -> String {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f.string(from: date)
    }
}
