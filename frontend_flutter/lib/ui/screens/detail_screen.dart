import 'dart:io';
import 'package:flutter/material.dart';
import 'package:frontend_flutter/data/models.dart';
import 'package:frontend_flutter/core/config.dart';
import 'package:frontend_flutter/ui/widgets/cached_image.dart';
import 'package:http/http.dart' as http;
import 'package:path_provider/path_provider.dart';
import 'package:frontend_flutter/data/api_client.dart';

class DetailScreen extends StatelessWidget {
  final String imageUrl;
  final MediaItem? mediaItem;
  final Function(File) onFindSimilar;
  final VoidCallback? onDelete;

  const DetailScreen({
    super.key,
    required this.imageUrl,
    this.mediaItem,
    required this.onFindSimilar,
    this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      extendBodyBehindAppBar: true,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        leading: IconButton(
          icon: const Icon(Icons.arrow_back_ios_new, color: Colors.white),
        onPressed: () => Navigator.pop(context),
        ),
        actions: [
          if (mediaItem != null)
            IconButton(
              icon: const Icon(Icons.delete_outline, color: Colors.redAccent),
              onPressed: () => _confirmDelete(context),
              tooltip: 'Delete Image',
            ),
        ],
      ),
      body: Stack(
        fit: StackFit.expand,
        children: [
          // Immersive Image
          InteractiveViewer(
            minScale: 1.0,
            maxScale: 4.0,
            child: Center(
              child: Hero(
                tag: imageUrl,
                child: CachedImage(
                  localPath: mediaItem?.localPath,
                  remoteUrl: imageUrl,
                  fit: BoxFit.contain,
                  loadingBuilder: (context, child, progress) {
                    if (progress == null) return child;
                    return const Center(
                      child: CircularProgressIndicator(color: Colors.white24),
                    );
                  },
                ),
              ),
            ),
          ),

          // Bottom Overlay
          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            child: Container(
              padding: const EdgeInsets.fromLTRB(24, 24, 24, 48),
              decoration: const BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.bottomCenter,
                  end: Alignment.topCenter,
                  colors: [
                    Colors.black87,
                    Colors.transparent,
                  ],
                ),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  // Info Button
                  IconButton(
                    onPressed: () => _showInfoSheet(context),
                    icon: const Icon(Icons.info_outline, color: Colors.white70, size: 28),
                    tooltip: 'Info',
                  ),

                  // Find Similar Button
                  ElevatedButton.icon(
                    onPressed: () => _handleFindSimilar(context),
                    icon: const Icon(Icons.image_search, size: 20),
                    label: const Text('Find Similar'),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: Colors.white,
                      foregroundColor: Colors.black,
                      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(30),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  void _showInfoSheet(BuildContext context) {
    showModalBottomSheet(
      context: context,
      backgroundColor: const Color(0xFF1E1E1E),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => Container(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Image Details',
              style: Theme.of(context).textTheme.titleLarge?.copyWith(color: Colors.white),
            ),
            const SizedBox(height: 16),
            _buildInfoRow('Date', mediaItem?.createdAt ?? 'Unknown'),
            _buildInfoRow('Size', mediaItem != null ? '${(mediaItem!.sizeBytes / 1024 / 1024).toStringAsFixed(2)} MB' : 'Unknown'),
            _buildInfoRow('Dimensions', mediaItem != null ? '${mediaItem!.width} x ${mediaItem!.height}' : 'Unknown'),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }

  Widget _buildInfoRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          SizedBox(
            width: 100,
            child: Text(
              label,
              style: const TextStyle(color: Colors.white54),
            ),
          ),
          Text(
            value,
            style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w500),
          ),
        ],
      ),
    );
  }

  Future<void> _handleFindSimilar(BuildContext context) async {
    // Capture references before any async gap to avoid context issues.
    final navigator = Navigator.of(context);
    final scaffold = ScaffoldMessenger.of(context);
    final callback = onFindSimilar;

    try {
      // Show loading indicator
      showDialog(
        context: context,
        barrierDismissible: false,
        builder: (ctx) => const Center(child: CircularProgressIndicator()),
      );

      // Use local file if available, otherwise download
      File searchFile;
      if (mediaItem?.localPath != null && File(mediaItem!.localPath!).existsSync()) {
        searchFile = File(mediaItem!.localPath!);
      } else {
        final response = await http.get(Uri.parse(AppConfig.fixImageUrl(imageUrl)));
        final tempDir = await getTemporaryDirectory();
        searchFile = File('${tempDir.path}/temp_search_image.jpg');
        await searchFile.writeAsBytes(response.bodyBytes);
      }

      navigator.pop(); // Dismiss loading
      navigator.pop(); // Close detail screen
      callback(searchFile); // Trigger search on Home
    } catch (e) {
      try {
        navigator.pop(); // Dismiss loading
      } catch (_) {}
      scaffold.showSnackBar(
        SnackBar(content: Text('Error finding similar: $e')),
      );
    }
  }

  Future<void> _confirmDelete(BuildContext context) async {
    if (mediaItem == null) return;
    
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: const Color(0xFF1E1E1E),
        title: const Text('Delete Image?', style: TextStyle(color: Colors.white)),
        content: const Text('This will permanently remove this image from the cloud gallery.',
            style: TextStyle(color: Colors.white70)),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Cancel', style: TextStyle(color: Colors.white54)),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Delete', style: TextStyle(color: Colors.redAccent)),
          ),
        ],
      ),
    );

    if (confirmed == true && context.mounted) {
      try {
        final api = ApiClient();
        await api.deleteImagesBatch([mediaItem!.id]);
        api.close();
        
        if (context.mounted) {
          Navigator.pop(context); // Close detail screen
          onDelete?.call(); // Notify home screen
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Image deleted')),
          );
        }
      } catch (e) {
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Delete failed: $e'), backgroundColor: Colors.redAccent),
          );
        }
      }
    }
  }
}
