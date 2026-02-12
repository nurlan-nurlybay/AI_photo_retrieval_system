import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:photo_manager/photo_manager.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:workmanager/workmanager.dart' as wm;

import 'package:frontend_flutter/core/category_cache_service.dart';
import 'package:frontend_flutter/data/api_client.dart';

/// Background task name.
const kPhotoWatcherTask = 'com.photoRetrieval.photoWatcher';

/// Key used to store the timestamp of the last checked photo.
const _kLastCheckedKey = 'photo_watcher_last_checked_ms';

/// Key for the enabled/disabled toggle.
const kPhotoWatcherEnabledKey = 'photo_watcher_enabled';

/// Notification channel (Android).
const _channelId = 'photo_classification';
const _channelName = 'Photo Classification';
const _channelDesc = 'Notifications about newly detected photo categories';

/// Simple keyword-based classifier that maps photo metadata to a category.
/// In production this would call the AI backend; here we use filename heuristics
/// so the feature works fully offline.
String? _classifyFilename(String path) {
  final lower = path.toLowerCase();
  // Recipe / food signals
  if (lower.contains('recipe') ||
      lower.contains('food') ||
      lower.contains('dish') ||
      lower.contains('cook') ||
      lower.contains('meal')) {
    return 'recipe';
  }
  // Screenshot signals
  if (lower.contains('screenshot') || lower.contains('screen_shot')) {
    return 'screenshot';
  }
  return null;
}

/// Messages shown per detected category.
const _categoryMessages = <String, String>{
  'recipe':
      'You took a photo of a recipe! Check out all recipes in your gallery in our app.',
  'screenshot':
      'New screenshot detected. View all your screenshots in our app.',
};

// ---------------------------------------------------------------------------
// Notification helpers
// ---------------------------------------------------------------------------

final FlutterLocalNotificationsPlugin _notifications =
    FlutterLocalNotificationsPlugin();

Future<void> initNotifications() async {
  const androidSettings = AndroidInitializationSettings('@mipmap/ic_launcher');
  const iosSettings = DarwinInitializationSettings(
    requestAlertPermission: true,
    requestBadgePermission: true,
    requestSoundPermission: true,
  );
  const settings =
      InitializationSettings(android: androidSettings, iOS: iosSettings);
  await _notifications.initialize(settings);

  // iOS: enable foreground notification presentation (banner + sound)
  await _notifications
      .resolvePlatformSpecificImplementation<
          IOSFlutterLocalNotificationsPlugin>()
      ?.requestPermissions(alert: true, badge: true, sound: true);
}

/// Public API for showing a local notification (used by settings debug button).
Future<void> showAppNotification(String title, String body) =>
    _showNotification(title, body);

Future<void> _showNotification(String title, String body) async {
  const androidDetails = AndroidNotificationDetails(
    _channelId,
    _channelName,
    channelDescription: _channelDesc,
    importance: Importance.high,
    priority: Priority.high,
  );
  const iosDetails = DarwinNotificationDetails(
    presentAlert: true,
    presentBanner: true,
    presentSound: true,
  );
  const details =
      NotificationDetails(android: androidDetails, iOS: iosDetails);

  await _notifications.show(
    DateTime.now().millisecondsSinceEpoch ~/ 1000, // unique id
    title,
    body,
    details,
  );
}

// ---------------------------------------------------------------------------
// Background callback (runs in isolate)
// ---------------------------------------------------------------------------

@pragma('vm:entry-point')
void callbackDispatcher() {
  wm.Workmanager().executeTask((taskName, inputData) async {
    if (taskName != kPhotoWatcherTask) return true;

    try {
      // Check if user has disabled notifications.
      final prefs = await SharedPreferences.getInstance();
      final enabled = prefs.getBool(kPhotoWatcherEnabledKey) ?? false;
      if (!enabled) return true;

      // Ensure we have gallery permission (granted at app start).
      final perm = await PhotoManager.requestPermissionExtend();
      if (!perm.isAuth) return true;

      // Get the timestamp of the last photo we checked.
      final lastCheckedMs = prefs.getInt(_kLastCheckedKey) ?? 0;
      final lastChecked =
          DateTime.fromMillisecondsSinceEpoch(lastCheckedMs);

      // Fetch recent photos created after our last check.
      final albums = await PhotoManager.getAssetPathList(
        type: RequestType.image,
        filterOption: FilterOptionGroup(
          createTimeCond: DateTimeCond(
            min: lastChecked,
            max: DateTime.now(),
          ),
          orders: [
            const OrderOption(type: OrderOptionType.createDate, asc: false),
          ],
        ),
      );

      if (albums.isEmpty) return true;
      final recent = await albums.first.getAssetListRange(start: 0, end: 20);

      if (recent.isEmpty) return true;

      // Update the last-checked timestamp to NOW.
      await prefs.setInt(
          _kLastCheckedKey, DateTime.now().millisecondsSinceEpoch);

      // Init notifications plugin inside the isolate.
      await initNotifications();

      // Classify each new photo.
      for (final asset in recent) {
        // Skip photos we already saw.
        if (asset.createDateTime.millisecondsSinceEpoch <= lastCheckedMs) {
          continue;
        }

        final file = await asset.file;
        if (file == null) continue;

        final category = _classifyFilename(file.path);
        if (category != null && _categoryMessages.containsKey(category)) {
          await _showNotification(
            'New photo detected',
            _categoryMessages[category]!,
          );
          // One notification per background run is enough.
          break;
        }
      }

      // --- AI-powered category refresh ---
      // Check if new photos were synced since last category cache.
      final needsCategoryRefresh =
          await CategoryCacheService.isRefreshNeeded();
      if (needsCategoryRefresh) {
        // Only attempt if backend is reachable
        final api = ApiClient();
        try {
          final online = await api.checkHealth();
          if (online) {
            // Notify user that categorization is starting
            final newCount = await CategoryCacheService.newPhotoCount();
            if (newCount > 0) {
              await _showNotification(
                'Categorizing photos',
                'Categorizing $newCount new photo${newCount == 1 ? '' : 's'}...',
              );
            }

            final result =
                await CategoryCacheService.refreshCategories();
            if (result.newPhotoCounts.isNotEmpty) {
              final totalNew = result.newPhotoCounts.values
                  .reduce((a, b) => a + b);
              final categoryNames =
                  result.newPhotoCounts.keys.take(3).join(', ');
              await _showNotification(
                'Photos categorized',
                '$totalNew new photo${totalNew == 1 ? '' : 's'} categorized into $categoryNames',
              );
            }
          }
        } finally {
          api.close();
        }
      }
    } catch (_) {
      // Swallow errors in background – don't crash the task.
    }

    return true;
  });
}

// ---------------------------------------------------------------------------
// Registration (called from main)
// ---------------------------------------------------------------------------

Future<void> initPhotoWatcher() async {
  await wm.Workmanager().initialize(callbackDispatcher, isInDebugMode: false);

  final prefs = await SharedPreferences.getInstance();
  final enabled = prefs.getBool(kPhotoWatcherEnabledKey) ?? false;
  if (enabled) {
    await registerPhotoWatcher();
  }
}

Future<void> registerPhotoWatcher() async {
  // Seed the "last checked" to now so we only look at future photos.
  final prefs = await SharedPreferences.getInstance();
  if (!prefs.containsKey(_kLastCheckedKey)) {
    await prefs.setInt(_kLastCheckedKey, DateTime.now().millisecondsSinceEpoch);
  }

  await wm.Workmanager().registerPeriodicTask(
    kPhotoWatcherTask,
    kPhotoWatcherTask,
    frequency: const Duration(minutes: 15),
    constraints: wm.Constraints(
      networkType: wm.NetworkType.notRequired,
    ),
    existingWorkPolicy: wm.ExistingPeriodicWorkPolicy.replace,
  );
}

Future<void> cancelPhotoWatcher() async {
  await wm.Workmanager().cancelByUniqueName(kPhotoWatcherTask);
}
