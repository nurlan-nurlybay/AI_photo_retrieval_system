import 'dart:io';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import '../../data/api_client.dart';
import '../../data/models.dart';
import 'package:frontend_flutter/data/api_client.dart';
import 'package:frontend_flutter/data/models.dart';


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

  Future<void> _pickImages() async {
    final res = await FilePicker.platform.pickFiles(
      allowMultiple: true,
      type: FileType.image,
    );
    if (res == null) return;
    setState(() {
      _picked = res.paths.whereType<String>().map((p) => File(p)).toList();
      _last = null;
    });
  }

  Future<void> _upload() async {
    if (_picked.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Please choose images first.')),
      );
      return;
    }
    setState(() => _busy = true);
    try {
      final resp = await _api.uploadImages(files: _picked, dedup: true);
      setState(() => _last = resp);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Saved: ${resp.summary.saved}, Duplicates: ${resp.summary.duplicates}, Failed: ${resp.summary.failed}')),
      );
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Upload failed: $e')),
      );
    } finally {
      setState(() => _busy = false);
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
            padding: EdgeInsets.all(12.0),
            child: LinearProgressIndicator(),
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
