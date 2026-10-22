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

    @Test("HearthItem decodes server JSON with nulls")
    func hearthItemDecoding() throws {
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
          "assigned_by": null,
          "will_complete_by": null,
          "completed_by": null,
          "completed_at": null,
          "created_at": "2026-10-22T19:30:00.000Z",
          "updated_at": "2026-10-22T19:30:00.000Z"
        }
        """#.utf8)
        let item = try JSONDecoder().decode(HearthItem.self, from: json)
        #expect(item.id == "01HFXYZ")
        #expect(item.title == "take out trash")
        #expect(item.state == "open")
        #expect(item.notes == nil)
        #expect(item.claimedBy == nil)
    }

    @Test("BacklogResponse decodes an empty list and a populated one")
    func backlogDecoding() throws {
        let empty = try JSONDecoder().decode(BacklogResponse.self, from: Data(#"{"items":[]}"#.utf8))
        #expect(empty.items.isEmpty)

        let populated = try JSONDecoder().decode(BacklogResponse.self, from: Data(#"""
        {
          "items": [
            {
              "id": "a", "list_id": "l", "creator_id": "u", "title": "t",
              "notes": null, "state": "open",
              "claimed_by": null, "claimed_at": null,
              "assigned_by": null, "will_complete_by": null,
              "completed_by": null, "completed_at": null,
              "created_at": "now", "updated_at": "now"
            }
          ]
        }
        """#.utf8))
        #expect(populated.items.count == 1)
        #expect(populated.items.first?.title == "t")
    }
}
