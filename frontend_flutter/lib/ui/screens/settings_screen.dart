import 'package:flutter/material.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:frontend_flutter/core/category_cache_service.dart';
import 'package:frontend_flutter/core/photo_watcher.dart';
import 'package:frontend_flutter/data/api_client.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  bool _galleryAccess = true;
  bool _pushNotifications = false;
  bool? _backendOnline;

  @override
  void initState() {
    super.initState();
    _checkBackend();
    _loadPushPref();
  }

  Future<void> _checkBackend() async {
    final online = await ApiClient().checkHealth();
    if (mounted) setState(() => _backendOnline = online);
  }

  Future<void> _loadPushPref() async {
    final prefs = await SharedPreferences.getInstance();
    if (mounted) {
      setState(() {
        _pushNotifications = prefs.getBool(kPhotoWatcherEnabledKey) ?? false;
      });
    }
  }

  Future<void> _togglePushNotifications(bool value) async {
    final prefs = await SharedPreferences.getInstance();

    if (value) {
      // Request notification permission (iOS / Android 13+).
      var status = await Permission.notification.status;

      if (status.isDenied) {
        status = await Permission.notification.request();
      }

      if (status.isPermanentlyDenied) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: const Text(
                  'Notification permission denied. Enable in Settings.'),
              action: SnackBarAction(
                label: 'Open Settings',
                onPressed: openAppSettings,
              ),
            ),
          );
        }
        return;
      }

      if (!status.isGranted) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
                content: Text('Notification permission denied.')),
          );
        }
        return;
      }

      await prefs.setBool(kPhotoWatcherEnabledKey, true);
      try {
        await registerPhotoWatcher();
      } catch (_) {
        // Workmanager not supported on this platform.
      }
    } else {
      await prefs.setBool(kPhotoWatcherEnabledKey, false);
      await cancelPhotoWatcher();
    }

    if (mounted) setState(() => _pushNotifications = value);
  }

  Future<void> _testCategoryNotification() async {
    // 1. Init notifications (requests iOS permission + enables foreground banners)
    await initNotifications();

    // 2. Check backend
    final api = ApiClient();
    final online = await api.checkHealth();
    api.close();

    if (!online) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Backend is offline')),
        );
      }
      return;
    }

    // 3. "Starting" push notification (iOS banner)
    final newCount = await CategoryCacheService.newPhotoCount();
    await showAppNotification(
      'Categorizing photos',
      'Categorizing $newCount new photo${newCount == 1 ? '' : 's'}...',
    );

    // 4. Run refresh + "done" push notification
    final result = await CategoryCacheService.refreshCategories();
    if (result.newPhotoCounts.isNotEmpty) {
      final totalNew =
          result.newPhotoCounts.values.reduce((a, b) => a + b);
      final names = result.newPhotoCounts.keys.take(3).join(', ');
      await showAppNotification(
        'Photos categorized',
        '$totalNew new photo${totalNew == 1 ? '' : 's'} categorized into $names',
      );
    } else {
      await showAppNotification(
        'Categories up to date',
        'No new photos found in any category',
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF121212),
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        children: [
          _sectionHeader('Permissions'),
          SwitchListTile(
            title: const Text('Gallery Access',
                style: TextStyle(color: Colors.white)),
            subtitle: const Text('Allow access to photo library',
                style: TextStyle(color: Colors.white54)),
            value: _galleryAccess,
            activeColor: Colors.white,
            onChanged: (v) => setState(() => _galleryAccess = v),
          ),
          ListTile(
            title: const Text('Request Permission',
                style: TextStyle(color: Colors.white)),
            trailing: OutlinedButton(
              style: OutlinedButton.styleFrom(
                foregroundColor: Colors.white,
                side: const BorderSide(color: Colors.white24),
              ),
              onPressed: () {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Permission requested')),
                );
              },
              child: const Text('Request'),
            ),
          ),
          const Divider(color: Colors.white10),
          _sectionHeader('Notifications'),
          SwitchListTile(
            title: const Text('Photo Classification Alerts',
                style: TextStyle(color: Colors.white)),
            subtitle: const Text(
                'Get notified when new photos are categorized',
                style: TextStyle(color: Colors.white54)),
            value: _pushNotifications,
            activeColor: Colors.white,
            onChanged: _togglePushNotifications,
          ),
          ListTile(
            title: const Text('Test Category Notification',
                style: TextStyle(color: Colors.orangeAccent)),
            leading: const Icon(Icons.bug_report, color: Colors.orangeAccent),
            onTap: _testCategoryNotification,
          ),
          const Divider(color: Colors.white10),
          _sectionHeader('About'),
          ListTile(
            title:
                const Text('Version', style: TextStyle(color: Colors.white)),
            trailing: const Text('1.0.0',
                style: TextStyle(color: Colors.white54)),
          ),
          ListTile(
            title: const Text('Backend Status',
                style: TextStyle(color: Colors.white)),
            trailing: _backendOnline == null
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                        strokeWidth: 2, color: Colors.white54),
                  )
                : Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Container(
                        width: 8,
                        height: 8,
                        decoration: BoxDecoration(
                          color: _backendOnline! ? Colors.green : Colors.red,
                          shape: BoxShape.circle,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Text(
                        _backendOnline! ? 'Online' : 'Offline',
                        style: TextStyle(
                          color: _backendOnline! ? Colors.green : Colors.red,
                        ),
                      ),
                    ],
                  ),
          ),
        ],
      ),
    );
  }

  Widget _sectionHeader(String title) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
      child: Text(
        title,
        style: const TextStyle(
          color: Colors.white54,
          fontSize: 13,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.5,
        ),
      ),
    );
  }
}
