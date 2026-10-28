import Foundation

/// Which chores the backlog is currently showing.
/// - `.all`: everything the user can see (claimed by anyone + unclaimed).
/// - `.member(id)`: only chores claimed by that household member; unclaimed
///   chores are hidden (filter means "show me what this person has").
enum BacklogFilter: Equatable {
    case all
    case member(String)

    var selectedMemberID: String? {
        if case .member(let id) = self { return id }
        return nil
    }
}
