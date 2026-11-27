import 'package:calango_theme/calango_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('light theme maps the brand tokens', () {
    final theme = CalangoTheme.light();

    expect(theme.brightness, Brightness.light);
    expect(theme.colorScheme.primary, CalangoColors.postit);
    expect(theme.colorScheme.onPrimary, CalangoColors.ink);
    expect(theme.colorScheme.surface, CalangoColors.paper);
    expect(theme.scaffoldBackgroundColor, CalangoColors.paper);
    expect(theme.textTheme.bodyMedium?.fontFamily, CalangoFonts.body);
    expect(theme.textTheme.headlineMedium?.fontFamily, CalangoFonts.display);
  });

  test('dark theme follows the desktop palette', () {
    final theme = CalangoTheme.dark();

    expect(theme.brightness, Brightness.dark);
    expect(theme.colorScheme.surface, CalangoColors.night);
    expect(theme.colorScheme.onSurface, CalangoColors.paper);
    expect(theme.colorScheme.surfaceContainerHighest, CalangoColors.nightCard);
    expect(theme.colorScheme.error, CalangoColors.tangerine);
  });

  test('component shapes use the token radii', () {
    final theme = CalangoTheme.light();
    final card = theme.cardTheme.shape! as RoundedRectangleBorder;
    final dialog = theme.dialogTheme.shape! as RoundedRectangleBorder;

    expect(
      card.borderRadius,
      const BorderRadius.all(Radius.circular(CalangoRadii.md)),
    );
    expect(
      dialog.borderRadius,
      const BorderRadius.all(Radius.circular(CalangoRadii.lg)),
    );
  });

  test('generated motion and shadow tokens', () {
    expect(CalangoMotion.fast, const Duration(milliseconds: 160));
    expect(CalangoMotion.easePaper.transform(1), closeTo(1, 1e-9));
    expect(CalangoShadows.paper, hasLength(2));
    expect(CalangoRadii.tileFraction, 0.28);
  });

  testWidgets('a MaterialApp renders with both themes', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: CalangoTheme.light(),
        darkTheme: CalangoTheme.dark(),
        themeMode: ThemeMode.dark,
        home: const Scaffold(body: Card(child: Text('Calango'))),
      ),
    );

    final context = tester.element(find.text('Calango'));
    expect(Theme.of(context).colorScheme.surface, CalangoColors.night);
    expect(DefaultTextStyle.of(context).style.fontFamily, CalangoFonts.body);
  });
}
