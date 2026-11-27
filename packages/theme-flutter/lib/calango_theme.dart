/// Calango theme for Flutter.
///
/// Every value comes from `lib/src/generated/tokens.dart`, generated from
/// `packages/tokens/tokens.json` by `scripts/generate-tokens.mjs`. This file
/// only maps tokens to Material roles; it must not introduce raw colors.
library;

import 'package:flutter/material.dart';

import 'src/generated/tokens.dart';

export 'src/generated/tokens.dart';

/// Light and dark [ThemeData] for the Calango apps.
abstract final class CalangoTheme {
  /// Paper background with ink text.
  static ThemeData light() => _theme(lightScheme);

  /// Night background with paper text, as in the desktop app.
  static ThemeData dark() => _theme(darkScheme);

  static const ColorScheme lightScheme = ColorScheme(
    brightness: Brightness.light,
    primary: CalangoColors.postit,
    onPrimary: CalangoColors.ink,
    primaryContainer: CalangoColors.postitLight,
    onPrimaryContainer: CalangoColors.ink,
    secondary: CalangoColors.leaf,
    onSecondary: CalangoColors.ink,
    secondaryContainer: CalangoColors.lime,
    onSecondaryContainer: CalangoColors.ink,
    tertiary: CalangoColors.tangerine,
    onTertiary: CalangoColors.ink,
    error: CalangoColors.clayDeep,
    onError: CalangoColors.paper,
    surface: CalangoColors.paper,
    onSurface: CalangoColors.ink,
    onSurfaceVariant: CalangoColors.inkSoft,
    surfaceContainerHighest: CalangoColors.sand,
    outline: CalangoColors.inkSoft,
    outlineVariant: CalangoColors.sand,
    shadow: CalangoColors.bark,
    inverseSurface: CalangoColors.night,
    onInverseSurface: CalangoColors.paper,
    inversePrimary: CalangoColors.postitDeep,
  );

  static const ColorScheme darkScheme = ColorScheme(
    brightness: Brightness.dark,
    primary: CalangoColors.postit,
    onPrimary: CalangoColors.ink,
    primaryContainer: CalangoColors.postitDeep,
    onPrimaryContainer: CalangoColors.ink,
    secondary: CalangoColors.lime,
    onSecondary: CalangoColors.ink,
    secondaryContainer: CalangoColors.moss,
    onSecondaryContainer: CalangoColors.paper,
    tertiary: CalangoColors.tangerine,
    onTertiary: CalangoColors.ink,
    error: CalangoColors.tangerine,
    onError: CalangoColors.ink,
    surface: CalangoColors.night,
    onSurface: CalangoColors.paper,
    onSurfaceVariant: CalangoColors.sand,
    surfaceContainer: CalangoColors.nightCard,
    surfaceContainerHigh: CalangoColors.nightCard,
    surfaceContainerHighest: CalangoColors.nightCard,
    outline: CalangoColors.inkSoft,
    outlineVariant: CalangoColors.inkSoft,
    shadow: CalangoColors.night,
    inverseSurface: CalangoColors.paper,
    onInverseSurface: CalangoColors.ink,
    inversePrimary: CalangoColors.postitDeep,
  );

  static ThemeData _theme(ColorScheme scheme) {
    final base = ThemeData(
      colorScheme: scheme,
      fontFamily: CalangoFonts.body,
      scaffoldBackgroundColor: scheme.surface,
    );
    const small = BorderRadius.all(Radius.circular(CalangoRadii.sm));
    const medium = BorderRadius.all(Radius.circular(CalangoRadii.md));
    const large = BorderRadius.all(Radius.circular(CalangoRadii.lg));
    const buttonShape = RoundedRectangleBorder(borderRadius: small);
    return base.copyWith(
      textTheme: _textTheme(base.textTheme),
      cardTheme: const CardThemeData(
        shape: RoundedRectangleBorder(borderRadius: medium),
      ),
      dialogTheme: const DialogThemeData(
        shape: RoundedRectangleBorder(borderRadius: large),
      ),
      bottomSheetTheme: const BottomSheetThemeData(
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(
            top: Radius.circular(CalangoRadii.lg),
          ),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(shape: buttonShape),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(shape: buttonShape),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(shape: buttonShape),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(shape: buttonShape),
      ),
      inputDecorationTheme: const InputDecorationTheme(
        border: OutlineInputBorder(borderRadius: small),
      ),
      chipTheme: const ChipThemeData(
        shape: RoundedRectangleBorder(borderRadius: small),
      ),
    );
  }

  /// Display and headline styles use the display face; the rest keep the body face.
  static TextTheme _textTheme(TextTheme base) {
    TextStyle? display(TextStyle? style, FontWeight weight) =>
        style?.copyWith(fontFamily: CalangoFonts.display, fontWeight: weight);
    return base.copyWith(
      displayLarge: display(base.displayLarge, FontWeight.w800),
      displayMedium: display(base.displayMedium, FontWeight.w800),
      displaySmall: display(base.displaySmall, FontWeight.w800),
      headlineLarge: display(base.headlineLarge, FontWeight.w700),
      headlineMedium: display(base.headlineMedium, FontWeight.w700),
      headlineSmall: display(base.headlineSmall, FontWeight.w700),
      titleLarge: display(base.titleLarge, FontWeight.w700),
    );
  }

  /// Monospace style for code, logs and identifiers.
  static const TextStyle mono = TextStyle(fontFamily: CalangoFonts.mono);
}
