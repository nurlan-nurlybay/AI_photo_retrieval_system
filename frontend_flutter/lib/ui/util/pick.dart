import 'dart:io';
import 'package:image_picker/image_picker.dart';

Future<bool> _hasCmd(String cmd) async {
  final r = await Process.run('which', [cmd]);
  return r.exitCode == 0;
}

Future<List<File>> _linuxPickMulti() async {
  // Try KDE first, then GTK. Both avoid DBus portals.
  if (await _hasCmd('kdialog')) {
    final r = await Process.run('kdialog', [
      '--getopenfilename',  // repeated to select multiple: use --getopenfilename with --multiple on some distros
      '--multiple',
      '--separate-output',
      '.', 'Images (*.png *.jpg *.jpeg *.gif *.webp *.bmp *.heic)'
    ]);
    if (r.exitCode == 0) {
      final lines = (r.stdout as String).trim().split('\n').where((s) => s.isNotEmpty);
      return lines.map((p) => File(p.replaceFirst('file://', ''))).toList();
    }
  }
  if (await _hasCmd('zenity')) {
    final r = await Process.run('zenity', [
      '--file-selection',
      '--multiple',
      '--separator=\n',
      '--title=Select images',
      '--file-filter=Images | *.png *.jpg *.jpeg *.gif *.webp *.bmp *.heic',
    ]);
    if (r.exitCode == 0) {
      final out = (r.stdout as String).trim();
      if (out.isNotEmpty) {
        return out.split('\n').map((p) => File(p)).toList();
      }
    }
  }
  // Fallback: tell user what to install
  stderr.writeln('No kdialog/zenity found. Install one to pick files without DBus.');
  return [];
}

Future<File?> _linuxPickSingle() async {
  if (await _hasCmd('kdialog')) {
    final r = await Process.run('kdialog', [
      '--getopenfilename',
      '.', 'Images (*.png *.jpg *.jpeg *.gif *.webp *.bmp *.heic)'
    ]);
    if (r.exitCode == 0) {
      final p = (r.stdout as String).trim();
      if (p.isNotEmpty) return File(p.replaceFirst('file://', ''));
    }
  }
  if (await _hasCmd('zenity')) {
    final r = await Process.run('zenity', [
      '--file-selection',
      '--title=Choose an image',
      '--file-filter=Images | *.png *.jpg *.jpeg *.gif *.webp *.bmp *.heic',
    ]);
    if (r.exitCode == 0) {
      final p = (r.stdout as String).trim();
      if (p.isNotEmpty) return File(p);
    }
  }
  stderr.writeln('No kdialog/zenity found. Install one to pick files without DBus.');
  return null;
}

Future<List<File>> pickImagesMulti() async {
  if (Platform.isAndroid || Platform.isIOS) {
    final xs = await ImagePicker().pickMultiImage(imageQuality: 92, maxWidth: 4096);
    return xs.map((x) => File(x.path)).toList();
  } else if (Platform.isLinux) {
    return _linuxPickMulti();
  } else {
    // macOS/Windows: native dialogs don’t need DBus; use file_selector if you want.
    // Minimal fallback: no picker.
    return [];
  }
}

Future<File?> pickImageSingle() async {
  if (Platform.isAndroid || Platform.isIOS) {
    final x = await ImagePicker().pickImage(source: ImageSource.gallery, imageQuality: 92, maxWidth: 4096);
    return x == null ? null : File(x.path);
  } else if (Platform.isLinux) {
    return _linuxPickSingle();
  } else {
    return null;
  }
}
