import 'package:flutter/material.dart';
import 'core/theme.dart';
import 'core/photo_watcher.dart';
import 'ui/widgets/app_shell.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  try {
    await initNotifications();
    await initPhotoWatcher();
  } catch (_) {
    // Workmanager may not be supported on all platforms (e.g. simulator).
  }

  runApp(MaterialApp(
    title: 'Photo Retrieval',
    debugShowCheckedModeBanner: false,
    theme: AppTheme.dark(),
    home: const AppShell(),
  ));
}
