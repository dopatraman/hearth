import Foundation
import Testing
@testable import HearthApp

@Suite("JSON decoding")
struct DecodingTests {
    @Test("HealthResponse decodes server JSON")
    func healthResponseDecoding() throws {
        let json = Data(#"{"status":"ok"}"#.utf8)
        let decoded = try JSONDecoder().decode(HealthResponse.self, from: json)
        #expect(decoded == HealthResponse(status: "ok"))
    }

    @Test("HearthChore decodes server JSON with nulls")
    func hearthChoreDecoding() throws {
        let json = Data(#"""
        {
          "id": "01HFXYZ",
          "list_id": "01HFLIST",
          "creator_id": "01HFUSR",
          "title": "take out trash",
          "notes": null,
          "state": "open",
          "claimed_by": null,
          "claimed_at": null,
          "will_complete_by": null,
          "completed_by": null,
          "completed_at": null,
          "created_at": "2026-10-22T19:30:00.000Z",
          "updated_at": "2026-10-22T19:30:00.000Z",
          "overdue": false
        }
        """#.utf8)
        let chore = try JSONDecoder().decode(HearthChore.self, from: json)
        #expect(chore.id == "01HFXYZ")
        #expect(chore.title == "take out trash")
        #expect(chore.state == "open")
        #expect(chore.notes == nil)
        #expect(chore.claimedBy == nil)
    }

    @Test("BacklogResponse decodes an empty list and a populated one")
    func backlogDecoding() throws {
        let empty = try JSONDecoder().decode(BacklogResponse.self, from: Data(#"{"chores":[]}"#.utf8))
        #expect(empty.chores.isEmpty)

        let populated = try JSONDecoder().decode(BacklogResponse.self, from: Data(#"""
        {
          "chores": [
            {
              "id": "a", "list_id": "l", "creator_id": "u", "title": "t",
              "notes": null, "state": "open",
              "claimed_by": null, "claimed_at": null,
              "will_complete_by": null,
              "completed_by": null, "completed_at": null,
              "created_at": "now", "updated_at": "now",
              "overdue": false
            }
          ]
        }
        """#.utf8))
        #expect(populated.chores.count == 1)
        #expect(populated.chores.first?.title == "t")
    }
}
