//
//  APIService.swift
//  TorrentSeeker
//
//  API service layer matching frontend API calls
//

import Foundation

enum APIError: Error {
    case unauthorized
    case invalidResponse
    case networkError(Error)
    case serverError(String)
}

class APIService {
    static let shared = APIService()

    // Update this to your backend URL
    private let baseURL = "http://localhost:8080/api"

    private var token: String? {
        UserDefaults.standard.string(forKey: "authToken")
    }

    // MARK: - Auth

    func login(username: String, password: String) async throws -> AuthResponse {
        let endpoint = "\(baseURL)/auth/login"
        var request = URLRequest(url: URL(string: endpoint)!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let body = ["username": username, "password": password]
        request.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode != 200 {
            let errorMessage = String(data: data, encoding: .utf8) ?? "Authentication failed"
            throw APIError.serverError(errorMessage)
        }

        return try JSONDecoder().decode(AuthResponse.self, from: data)
    }

    func register(username: String, password: String) async throws -> AuthResponse {
        let endpoint = "\(baseURL)/auth/register"
        var request = URLRequest(url: URL(string: endpoint)!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let body = ["username": username, "password": password]
        request.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode != 200 {
            let errorMessage = String(data: data, encoding: .utf8) ?? "Registration failed"
            throw APIError.serverError(errorMessage)
        }

        return try JSONDecoder().decode(AuthResponse.self, from: data)
    }

    // MARK: - Items

    func getItems() async throws -> [Item] {
        try await authFetch(endpoint: "\(baseURL)/items")
    }

    func addItem(text: String) async throws {
        let endpoint = "\(baseURL)/items"
        var request = try createAuthRequest(url: endpoint, method: "POST")

        let body = "text=\(text.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")"
        request.httpBody = body.data(using: .utf8)
        request.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")

        let (_, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }

        if httpResponse.statusCode != 200 {
            throw APIError.serverError("Failed to add item")
        }
    }

    func updateItem(id: Int, text: String) async throws {
        let endpoint = "\(baseURL)/items/\(id)"
        var request = try createAuthRequest(url: endpoint, method: "PUT")

        let body = "text=\(text.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")"
        request.httpBody = body.data(using: .utf8)
        request.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")

        let (_, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }
    }

    func deleteItem(id: Int) async throws {
        let endpoint = "\(baseURL)/items/\(id)"
        let request = try createAuthRequest(url: endpoint, method: "DELETE")

        let (_, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }
    }

    // MARK: - URLs

    func getURLs() async throws -> [URL] {
        try await authFetch(endpoint: "\(baseURL)/urls")
    }

    func addURL(url: String) async throws {
        let endpoint = "\(baseURL)/urls"
        var request = try createAuthRequest(url: endpoint, method: "POST")

        let body = "url=\(url.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")"
        request.httpBody = body.data(using: .utf8)
        request.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")

        let (_, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }
    }

    func updateURL(id: Int, url: String?, displayName: String?) async throws {
        let endpoint = "\(baseURL)/urls/\(id)"
        var request = try createAuthRequest(url: endpoint, method: "PUT")

        var params: [String] = []
        if let url = url {
            params.append("url=\(url.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")")
        }
        if let displayName = displayName {
            params.append("display_name=\(displayName.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")")
        }

        let body = params.joined(separator: "&")
        request.httpBody = body.data(using: .utf8)
        request.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")

        let (_, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }
    }

    func deleteURL(id: Int) async throws {
        let endpoint = "\(baseURL)/urls/\(id)"
        let request = try createAuthRequest(url: endpoint, method: "DELETE")

        let (_, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }
    }

    // MARK: - Matches

    func getMatches() async throws -> [Match] {
        try await authFetch(endpoint: "\(baseURL)/matches")
    }

    func softDeleteMatch(id: Int) async throws {
        let endpoint = "\(baseURL)/matches/\(id)"
        let request = try createAuthRequest(url: endpoint, method: "DELETE")

        let (_, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }
    }

    func hardDeleteMatch(id: Int) async throws {
        let endpoint = "\(baseURL)/matches/\(id)?hard=true"
        let request = try createAuthRequest(url: endpoint, method: "DELETE")

        let (_, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }
    }

    // MARK: - Logs

    func getLogs(page: Int = 1) async throws -> LogsResponse {
        try await authFetch(endpoint: "\(baseURL)/logs?page=\(page)")
    }

    func clearLogs() async throws {
        let endpoint = "\(baseURL)/logs"
        let request = try createAuthRequest(url: endpoint, method: "DELETE")

        let (_, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }
    }

    // MARK: - Worker

    func triggerWorker() async throws -> WorkerStatusResponse {
        let endpoint = "\(baseURL)/trigger-worker"
        let request = try createAuthRequest(url: endpoint, method: "POST")

        let (data, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }

        return try JSONDecoder().decode(WorkerStatusResponse.self, from: data)
    }

    func testSMS() async throws -> String {
        let endpoint = "\(baseURL)/test-sms"
        let request = try createAuthRequest(url: endpoint, method: "POST")

        let (data, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }

        if httpResponse.statusCode != 200 {
            let errorMessage = String(data: data, encoding: .utf8) ?? "Failed to send test SMS"
            throw APIError.serverError(errorMessage)
        }

        if let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
           let message = json["message"] as? String {
            return message
        }

        return "Test SMS sent successfully"
    }

    // MARK: - Helper Methods

    private func authFetch<T: Decodable>(endpoint: String) async throws -> T {
        let request = try createAuthRequest(url: endpoint, method: "GET")

        let (data, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }

        if httpResponse.statusCode != 200 {
            throw APIError.serverError("Request failed with status \(httpResponse.statusCode)")
        }

        return try JSONDecoder().decode(T.self, from: data)
    }

    private func createAuthRequest(url: String, method: String) throws -> URLRequest {
        guard let token = token else {
            throw APIError.unauthorized
        }

        var request = URLRequest(url: Foundation.URL(string: url)!)
        request.httpMethod = method
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        return request
    }
}
