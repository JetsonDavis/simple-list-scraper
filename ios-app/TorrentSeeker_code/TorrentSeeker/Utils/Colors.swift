//
//  Colors.swift
//  TorrentSeeker
//
//  Color constants matching the CSS design
//

import SwiftUI

extension Color {
    // Azure blue theme colors
    static let azureBlue = Color(red: 0.0, green: 0.47, blue: 0.831)        // #0078D4
    static let azureBlueDark = Color(red: 0.0, green: 0.353, blue: 0.62)    // #005A9E
    static let azureBlueLight = Color(red: 0.902, green: 0.949, blue: 0.98) // #E6F2FA
    static let azureBlueHover = Color(red: 0.063, green: 0.431, blue: 0.745) // #106EBE

    // Border and text colors
    static let borderColor = Color(red: 0.882, green: 0.882, blue: 0.882)   // #E1E1E1
    static let textSecondary = Color(red: 0.376, green: 0.369, blue: 0.361) // #605E5C

    // Background colors
    static let backgroundLight = Color(red: 0.953, green: 0.949, blue: 0.945) // #F3F2F1
    static let backgroundGray = Color(red: 0.98, green: 0.98, blue: 0.98)     // #FAFAFA

    // Status colors
    static let errorRed = Color(red: 0.82, green: 0.2, blue: 0.22)           // #D13438
    static let warningOrange = Color.orange
}
