import 'dart:io';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:frontend_flutter/core/photo_sync.dart';
import 'package:frontend_flutter/data/api_client.dart';
import 'package:frontend_flutter/data/models.dart';
import 'package:frontend_flutter/ui/util/pick.dart';


class ImageUploadScreen extends StatefulWidget {
  const ImageUploadScreen({super.key});

  @override
  State<ImageUploadScreen> createState() => _ImageUploadScreenState();
}

class _ImageUploadScreenState extends State<ImageUploadScreen> {
  final _api = ApiClient();
  List<File> _picked = [];
  bool _busy = false;
  UploadResponseModel? _last;
  String? _batchProgress;

Future<void> _pickImages() async {
  try {
    final files = await pickImagesMulti();
    if (!mounted) return;
    setState(() {
      _picked = files;
      _last = null;
    });
    if (files.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('No images selected')));
    } else {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Selected ${files.length} images')));
    }
  } catch (e) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Error picking images: $e')));
  }
}

  Future<void> _upload() async {
    if (_picked.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Please choose images first.')),
      );
      return;
    }
    setState(() { _busy = true; _batchProgress = null; });

    const batchSize = 10;
    int totalSaved = 0, totalDups = 0, totalFailed = 0;
    final localPaths = _picked.map((f) => f.path).toList();
    final totalBatches = (_picked.length / batchSize).ceil();

    try {
      for (int i = 0; i < _picked.length; i += batchSize) {
        final end = (i + batchSize).clamp(0, _picked.length);
        final batchNum = i ~/ batchSize + 1;
        setState(() => _batchProgress = 'Uploading batch $batchNum / $totalBatches…');

        final resp = await _api.uploadImages(
          files: _picked.sublist(i, end),
          localPaths: localPaths.sublist(i, end),
          dedup: true,
        );
        totalSaved   += resp.summary.saved;
        totalDups    += resp.summary.duplicates;
        totalFailed  += resp.summary.failed;
        setState(() => _last = resp);
      }

      if (totalSaved > 0) {
        PhotoSyncService.instance.notifyNewUploads(totalSaved);
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Saved: $totalSaved, Duplicates: $totalDups, Failed: $totalFailed')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Upload failed: $e')),
        );
      }
    } finally {
      setState(() { _busy = false; _batchProgress = null; });
    }
  }

  @override
  Widget build(BuildContext context) {
    final black = Theme.of(context).colorScheme.primary;
    return Scaffold(
      appBar: AppBar(title: const Text('Image Upload')),
      body: Column(
        children: [
          const SizedBox(height: 12),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              ElevatedButton.icon(
                onPressed: _busy ? null : _pickImages,
                icon: const Icon(Icons.folder_open),
                label: const Text('Choose Images'),
              ),
              const SizedBox(width: 12),
              ElevatedButton.icon(
                onPressed: _busy ? null : _upload,
                icon: const Icon(Icons.cloud_upload),
                label: const Text('Upload'),
              ),
            ],
          ),
          if (_busy) const Padding(
            padding: EdgeInsets.symmetric(horizontal: 12, vertical: 4),
            child: LinearProgressIndicator(),
          ),
          if (_batchProgress != null) Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 2),
            child: Text(_batchProgress!, style: const TextStyle(fontSize: 12, color: Colors.white54)),
          ),
          Expanded(
            child: _picked.isEmpty
                ? const Center(child: Text('No images selected'))
                : GridView.builder(
                    padding: const EdgeInsets.all(12),
                    gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                      crossAxisCount: 3, mainAxisSpacing: 8, crossAxisSpacing: 8,
                    ),
                    itemCount: _picked.length,
                    itemBuilder: (_, i) => ClipRRect(
                      borderRadius: BorderRadius.circular(8),
                      child: Image.file(_picked[i], fit: BoxFit.cover),
                    ),
                  ),
          ),
          if (_last != null)
            Container(
              width: double.infinity,
              color: Colors.black12,
              padding: const EdgeInsets.all(12),
              child: Text(
                'Summary — total: ${_last!.summary.total}, saved: ${_last!.summary.saved}, '
                'duplicates: ${_last!.summary.duplicates}, failed: ${_last!.summary.failed}',
                style: TextStyle(color: black),
              ),
            ),
        ],
      ),
    );
  }
}
