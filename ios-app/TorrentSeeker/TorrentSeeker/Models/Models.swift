//
//  Models.swift
//  TorrentSeeker
//
//  Data models matching the React frontend
//

import Foundation

struct Item: Codable, Identifiable {
    let id: Int
    let text: String
}

struct SiteURL: Codable, Identifiable {
    let id: Int
    let url: String
    let displayName: String?
    let config: String?

    enum CodingKeys: String, CodingKey {
        case id, url, config
        case displayName = "display_name"
    }
}

struct Match: Codable, Identifiable {
    let id: Int
    let item: String
    let url: String
    let site: String
    let torrentText: String?
    let magnetLink: String?
    let fileSize: String?
    let created: String

    enum CodingKeys: String, CodingKey {
        case id, item, url, site, created
        case torrentText = "torrent_text"
        case magnetLink = "magnet_link"
        case fileSize = "file_size"
    }
}

struct Log: Codable, Identifiable {
    let id: Int
    let timestamp: String
    let description: String
    let success: Bool
}

struct LogsResponse: Codable {
    let logs: [Log]
    let page: Int
    let pageSize: Int
    let total: Int
    let totalPages: Int

    enum CodingKeys: String, CodingKey {
        case logs, page, total
        case pageSize = "page_size"
        case totalPages = "total_pages"
    }
}

struct AuthResponse: Codable {
    let token: String
    let username: String
}

struct WorkerStatusResponse: Codable {
    let running: Bool
    let status: String?
    let message: String?
}

struct WebSocketMessage: Codable {
    let type: String
    let status: String?
    let message: String?
    let match: Match?
    let log: Log?
}
