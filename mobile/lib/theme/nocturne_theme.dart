import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

/// Design tokens ported from the Claude Design "Nocturne" system
/// (`systems/nocturne/styles.css` in the downloaded mock-up) - see
/// documentations/features/2026-07-21-nocturne-redesign.md.
class NocturneColors {
  NocturneColors._();

  static const bg = Color(0xFF161826);
  static const surface = Color(0xFF232532);
  static const text = Color(0xFFE9E9ED);
  static const accent = Color(0xFF9184D9);
  static const accent2 = Color(0xFFA7A1DB);
  static const divider = Color(0x29E9E9ED); // color-text at 16% alpha

  static const accent100 = Color(0xFFF5F4FF);
  static const accent300 = Color(0xFFD2CEFD);
  static const accent700 = Color(0xFF5D5294);
  static const accent800 = Color(0xFF423A6A);

  static const neutral100 = Color(0xFFF3F5FE);
  static const neutral800 = Color(0xFF3F424D);

  /// Nocturne has no separate warning/error semantic tokens - accent-300
  /// stands in for warnings (it's already the "notice" tint against the dark
  /// surface), and a standard accessible red covers errors, same reasoning
  /// as the plan's B1 note.
  static const warning = accent300;
  static const error = Color(0xFFE5787A);
}

class NocturneRadii {
  NocturneRadii._();

  static const sm = 4.0;
  static const md = 8.0;
  static const lg = 14.0;
}

ThemeData buildNocturneTheme() {
  final textTheme = GoogleFonts.interTextTheme(ThemeData.dark().textTheme).apply(
    bodyColor: NocturneColors.text,
    displayColor: NocturneColors.text,
  );

  final colorScheme = ColorScheme.fromSeed(
    seedColor: NocturneColors.accent,
    brightness: Brightness.dark,
    surface: NocturneColors.surface,
    onSurface: NocturneColors.text,
    primary: NocturneColors.accent,
    onPrimary: NocturneColors.bg,
    secondary: NocturneColors.accent2,
    error: NocturneColors.error,
  );

  return ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    colorScheme: colorScheme,
    scaffoldBackgroundColor: NocturneColors.bg,
    textTheme: textTheme,
    fontFamily: GoogleFonts.inter().fontFamily,
    appBarTheme: AppBarTheme(
      backgroundColor: NocturneColors.bg,
      foregroundColor: NocturneColors.text,
      elevation: 0,
      titleTextStyle: GoogleFonts.inter(
        color: NocturneColors.text,
        fontSize: 18,
        fontWeight: FontWeight.w500,
      ),
    ),
    cardTheme: CardThemeData(
      color: NocturneColors.surface,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(NocturneRadii.md),
        side: const BorderSide(color: NocturneColors.divider),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: NocturneColors.surface,
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(NocturneRadii.md),
        borderSide: const BorderSide(color: NocturneColors.divider),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(NocturneRadii.md),
        borderSide: const BorderSide(color: NocturneColors.divider),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(NocturneRadii.md),
        borderSide: const BorderSide(color: NocturneColors.accent, width: 1.5),
      ),
      labelStyle: TextStyle(color: NocturneColors.text.withValues(alpha: 0.7)),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: NocturneColors.accent,
        foregroundColor: NocturneColors.bg,
        textStyle: GoogleFonts.inter(fontWeight: FontWeight.w500),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(NocturneRadii.md),
        ),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: NocturneColors.accent,
        side: const BorderSide(color: NocturneColors.accent),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(NocturneRadii.md),
        ),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: NocturneColors.accent,
      ),
    ),
    floatingActionButtonTheme: const FloatingActionButtonThemeData(
      backgroundColor: NocturneColors.accent,
      foregroundColor: NocturneColors.bg,
    ),
    dividerTheme: const DividerThemeData(color: NocturneColors.divider, thickness: 1),
    listTileTheme: const ListTileThemeData(
      iconColor: NocturneColors.accent,
      textColor: NocturneColors.text,
    ),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: NocturneColors.surface,
      contentTextStyle: const TextStyle(color: NocturneColors.text),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(NocturneRadii.md),
      ),
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: NocturneColors.surface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(NocturneRadii.lg),
      ),
    ),
  );
}
