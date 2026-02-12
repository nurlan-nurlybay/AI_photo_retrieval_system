import 'package:flutter/material.dart';

import 'package:frontend_flutter/core/config.dart';
import 'package:frontend_flutter/data/api_client.dart';
import 'package:frontend_flutter/data/models.dart';
import 'package:frontend_flutter/ui/widgets/cached_image.dart';

/// Displays photos inside an album.
/// Returns the updated [PhotoAlbum] via Navigator.pop when going back.
class AlbumPhotosScreen extends StatefulWidget {
  final PhotoAlbum album;

  const AlbumPhotosScreen({super.key, required this.album});

  @override
  State<AlbumPhotosScreen> createState() => _AlbumPhotosScreenState();
}

class _AlbumPhotosScreenState extends State<AlbumPhotosScreen> {
  late PhotoAlbum _album;

  @override
  void initState() {
    super.initState();
    _album = widget.album;
  }

  Future<void> _addPhotos() async {
    final selected = await Navigator.push<List<AlbumPhoto>>(
      context,
      MaterialPageRoute(
        builder: (_) => _PhotoPickerScreen(existingIds: _album.photos.map((p) => p.id).toSet()),
      ),
    );
    if (selected != null && selected.isNotEmpty && mounted) {
      setState(() {
        _album = PhotoAlbum(
          id: _album.id,
          name: _album.name,
          photos: [..._album.photos, ...selected],
        );
      });
    }
  }

  void _removePhoto(int index) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: const Color(0xFF1E1E1E),
        title: const Text('Remove from Album', style: TextStyle(color: Colors.white)),
        content: const Text(
          'Remove this photo from the album? It won\'t be deleted from your library.',
          style: TextStyle(color: Colors.white70),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Cancel', style: TextStyle(color: Colors.white54)),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Remove', style: TextStyle(color: Colors.redAccent)),
          ),
        ],
      ),
    ).then((confirmed) {
      if (confirmed == true) {
        setState(() {
          final photos = List<AlbumPhoto>.from(_album.photos)..removeAt(index);
          _album = PhotoAlbum(id: _album.id, name: _album.name, photos: photos);
        });
      }
    });
  }

  void _openPhoto(AlbumPhoto photo) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => Scaffold(
          backgroundColor: Colors.black,
          extendBodyBehindAppBar: true,
          appBar: AppBar(
            backgroundColor: Colors.transparent,
            leading: IconButton(
              icon: const Icon(Icons.arrow_back_ios_new, color: Colors.white),
              onPressed: () => Navigator.pop(context),
            ),
          ),
          body: Center(
            child: InteractiveViewer(
              minScale: 1.0,
              maxScale: 4.0,
              child: CachedImage(
                localPath: photo.localPath,
                remoteUrl: photo.url,
                fit: BoxFit.contain,
              ),
            ),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: true,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop) Navigator.of(context).maybePop(_album);
      },
      child: Scaffold(
        backgroundColor: const Color(0xFF121212),
        appBar: AppBar(
          title: Text('${_album.name} (${_album.count})'),
          leading: IconButton(
            icon: const Icon(Icons.arrow_back),
            onPressed: () => Navigator.pop(context, _album),
          ),
        ),
        floatingActionButton: FloatingActionButton(
          backgroundColor: Colors.white12,
          onPressed: _addPhotos,
          child: const Icon(Icons.add_photo_alternate, color: Colors.white),
        ),
        body: _album.photos.isEmpty ? _buildEmptyState() : _buildGrid(),
      ),
    );
  }

  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.photo_library_outlined, size: 48, color: Colors.white24),
          const SizedBox(height: 16),
          const Text(
            'No photos in this album yet',
            style: TextStyle(color: Colors.white54, fontSize: 16),
          ),
          const SizedBox(height: 16),
          OutlinedButton.icon(
            onPressed: _addPhotos,
            icon: const Icon(Icons.add_photo_alternate),
            label: const Text('Add Photos'),
            style: OutlinedButton.styleFrom(
              foregroundColor: Colors.white,
              side: const BorderSide(color: Colors.white24),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildGrid() {
    return Padding(
      padding: const EdgeInsets.all(4),
      child: GridView.builder(
        itemCount: _album.photos.length,
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 3,
          mainAxisSpacing: 2,
          crossAxisSpacing: 2,
        ),
        itemBuilder: (context, index) {
          final photo = _album.photos[index];
          return GestureDetector(
            onTap: () => _openPhoto(photo),
            onLongPress: () => _removePhoto(index),
            child: ClipRRect(
              borderRadius: BorderRadius.circular(4),
              child: CachedImage(
                localPath: photo.localPath,
                remoteUrl: photo.thumbUrl.isNotEmpty ? photo.thumbUrl : photo.url,
                fit: BoxFit.cover,
              ),
            ),
          );
        },
      ),
    );
  }
}

/// Picker screen: loads all user images from the backend, lets user select multiple.
class _PhotoPickerScreen extends StatefulWidget {
  final Set<int> existingIds;

  const _PhotoPickerScreen({required this.existingIds});

  @override
  State<_PhotoPickerScreen> createState() => _PhotoPickerScreenState();
}

class _PhotoPickerScreenState extends State<_PhotoPickerScreen> {
  bool _isLoading = true;
  String? _error;
  List<MediaItem> _allPhotos = [];
  final Set<int> _selected = {};

  @override
  void initState() {
    super.initState();
    _loadPhotos();
  }

  Future<void> _loadPhotos() async {
    try {
      final photos = await ApiClient.listUserImages(
        userId: AppConfig.userId,
        limit: 200,
      );
      if (mounted) {
        setState(() {
          _allPhotos = photos.where((p) => !widget.existingIds.contains(p.id)).toList();
          _isLoading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isLoading = false;
        });
      }
    }
  }

  void _confirm() {
    final picked = _allPhotos
        .where((p) => _selected.contains(p.id))
        .map((p) => AlbumPhoto.fromMediaItem(p))
        .toList();
    Navigator.pop(context, picked);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF121212),
      appBar: AppBar(
        title: Text(_selected.isEmpty ? 'Select Photos' : '${_selected.length} selected'),
        actions: [
          if (_selected.isNotEmpty)
            TextButton(
              onPressed: _confirm,
              child: const Text('Add', style: TextStyle(fontSize: 16)),
            ),
        ],
      ),
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (_isLoading) {
      return const Center(child: CircularProgressIndicator(color: Colors.white));
    }
    if (_error != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.cloud_off, color: Colors.white24, size: 48),
            const SizedBox(height: 16),
            Text(_error!, style: const TextStyle(color: Colors.redAccent)),
            const SizedBox(height: 16),
            OutlinedButton(
              onPressed: () {
                setState(() {
                  _isLoading = true;
                  _error = null;
                });
                _loadPhotos();
              },
              style: OutlinedButton.styleFrom(
                foregroundColor: Colors.white,
                side: const BorderSide(color: Colors.white24),
              ),
              child: const Text('Retry'),
            ),
          ],
        ),
      );
    }
    if (_allPhotos.isEmpty) {
      return const Center(
        child: Text(
          'No photos available.\nUpload photos first.',
          textAlign: TextAlign.center,
          style: TextStyle(color: Colors.white54, fontSize: 16),
        ),
      );
    }

    return Padding(
      padding: const EdgeInsets.all(4),
      child: GridView.builder(
        itemCount: _allPhotos.length,
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 3,
          mainAxisSpacing: 2,
          crossAxisSpacing: 2,
        ),
        itemBuilder: (context, index) {
          final photo = _allPhotos[index];
          final isSelected = _selected.contains(photo.id);
          return GestureDetector(
            onTap: () {
              setState(() {
                if (isSelected) {
                  _selected.remove(photo.id);
                } else {
                  _selected.add(photo.id);
                }
              });
            },
            child: Stack(
              fit: StackFit.expand,
              children: [
                ClipRRect(
                  borderRadius: BorderRadius.circular(4),
                  child: CachedImage(
                    localPath: photo.localPath,
                    remoteUrl: photo.thumbUrl.isNotEmpty ? photo.thumbUrl : photo.url,
                    fit: BoxFit.cover,
                  ),
                ),
                if (isSelected)
                  Container(
                    decoration: BoxDecoration(
                      color: Colors.blue.withValues(alpha: 0.3),
                      borderRadius: BorderRadius.circular(4),
                      border: Border.all(color: Colors.blue, width: 3),
                    ),
                    child: const Align(
                      alignment: Alignment.topRight,
                      child: Padding(
                        padding: EdgeInsets.all(4),
                        child: Icon(Icons.check_circle, color: Colors.blue, size: 24),
                      ),
                    ),
                  ),
              ],
            ),
          );
        },
      ),
    );
  }
}
