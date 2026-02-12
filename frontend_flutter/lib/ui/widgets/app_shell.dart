import 'dart:async';

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:frontend_flutter/core/category_cache_service.dart';
import 'package:frontend_flutter/core/photo_sync.dart';
import 'package:frontend_flutter/core/photo_watcher.dart';
import 'package:frontend_flutter/ui/screens/home_screen.dart';
import 'package:frontend_flutter/ui/screens/categories_screen.dart';
import 'package:frontend_flutter/ui/screens/albums_screen.dart';
import 'package:frontend_flutter/ui/screens/settings_screen.dart';

class AppShell extends StatefulWidget {
  const AppShell({super.key});

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  int _selectedIndex = 0;
  StreamSubscription<PhotoSyncProgress>? _syncSub;

  @override
  void initState() {
    super.initState();
    _listenToSync();
  }

  @override
  void dispose() {
    _syncSub?.cancel();
    super.dispose();
  }

  /// After sync completes with new uploads, wait 30s for embeddings,
  /// then run category refresh and fire an iOS push notification.
  void _listenToSync() {
    _syncSub = PhotoSyncService.instance.progressStream.listen((progress) {
      if (progress.status == PhotoSyncStatus.completed &&
          progress.syncedPhotos > 0) {
        Future.delayed(const Duration(seconds: 30), () {
          if (mounted) _refreshAndNotify();
        });
      }
    });
  }

  Future<void> _refreshAndNotify() async {
    try {
      // Check if notifications are enabled
      final prefs = await SharedPreferences.getInstance();
      final enabled = prefs.getBool(kPhotoWatcherEnabledKey) ?? false;
      if (!enabled) return;

      final needsRefresh = await CategoryCacheService.isRefreshNeeded();
      if (!needsRefresh) return;

      await initNotifications();

      final newCount = await CategoryCacheService.newPhotoCount();
      if (newCount > 0) {
        await showAppNotification(
          'Categorizing photos',
          'Categorizing $newCount new photo${newCount == 1 ? '' : 's'}...',
        );
      }

      final result = await CategoryCacheService.refreshCategories();
      if (result.newPhotoCounts.isNotEmpty) {
        final totalNew =
            result.newPhotoCounts.values.reduce((a, b) => a + b);
        final names = result.newPhotoCounts.keys.take(3).join(', ');
        await showAppNotification(
          'Photos categorized',
          '$totalNew new photo${totalNew == 1 ? '' : 's'} categorized into $names',
        );
      }
    } catch (_) {
      // Don't crash the shell on notification errors
    }
  }

  static const _destinations = [
    NavigationDestination(
      icon: Icon(Icons.photo_library_outlined),
      selectedIcon: Icon(Icons.photo_library),
      label: 'Gallery',
    ),
    NavigationDestination(
      icon: Icon(Icons.category_outlined),
      selectedIcon: Icon(Icons.category),
      label: 'Categories',
    ),
    NavigationDestination(
      icon: Icon(Icons.photo_album_outlined),
      selectedIcon: Icon(Icons.photo_album),
      label: 'Albums',
    ),
    NavigationDestination(
      icon: Icon(Icons.settings_outlined),
      selectedIcon: Icon(Icons.settings),
      label: 'Settings',
    ),
  ];

  static const _railDestinations = [
    NavigationRailDestination(
      icon: Icon(Icons.photo_library_outlined),
      selectedIcon: Icon(Icons.photo_library),
      label: Text('Gallery'),
    ),
    NavigationRailDestination(
      icon: Icon(Icons.category_outlined),
      selectedIcon: Icon(Icons.category),
      label: Text('Categories'),
    ),
    NavigationRailDestination(
      icon: Icon(Icons.photo_album_outlined),
      selectedIcon: Icon(Icons.photo_album),
      label: Text('Albums'),
    ),
    NavigationRailDestination(
      icon: Icon(Icons.settings_outlined),
      selectedIcon: Icon(Icons.settings),
      label: Text('Settings'),
    ),
  ];

  final _screens = const [
    HomeScreen(),
    CategoriesScreen(),
    AlbumsScreen(),
    SettingsScreen(),
  ];

  void _onDestinationSelected(int index) {
    setState(() => _selectedIndex = index);
  }

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.of(context).size.width;
    final useRail = width >= 600;

    final body = IndexedStack(
      index: _selectedIndex,
      children: _screens,
    );

    if (useRail) {
      return Scaffold(
        body: Row(
          children: [
            NavigationRail(
              selectedIndex: _selectedIndex,
              onDestinationSelected: _onDestinationSelected,
              labelType: NavigationRailLabelType.all,
              destinations: _railDestinations,
            ),
            const VerticalDivider(width: 1, thickness: 1, color: Colors.white10),
            Expanded(child: body),
          ],
        ),
      );
    }

    return Scaffold(
      body: body,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _selectedIndex,
        onDestinationSelected: _onDestinationSelected,
        destinations: _destinations,
      ),
    );
  }
}
