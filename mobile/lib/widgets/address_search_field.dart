import 'package:flutter/material.dart';

import '../models/geocode_result.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';

/// Optional address search bar (see documentations/features/ entry) - sits
/// ABOVE a tap-to-pick map, never replacing it: this widget only calls back
/// with a chosen [GeocodeResult], the parent screen decides what to do with
/// it (same as it already does for a map tap).
class AddressSearchField extends StatefulWidget {
  final ApiClient api;
  final ValueChanged<GeocodeResult> onSelected;
  final String hintText;
  // Optional externally-owned controller - lets a parent screen programmatically
  // set the displayed text (e.g. a reverse-geocoded address after a map tap,
  // see route_request_screen.dart) without this widget losing its own typing/
  // search state. Caller owns disposal when provided.
  final TextEditingController? controller;
  const AddressSearchField({
    super.key,
    required this.api,
    required this.onSelected,
    this.hintText = 'Pretraga adrese (opciono)',
    this.controller,
  });

  @override
  State<AddressSearchField> createState() => _AddressSearchFieldState();
}

class _AddressSearchFieldState extends State<AddressSearchField> {
  late final TextEditingController _ctrl = widget.controller ?? TextEditingController();
  List<GeocodeResult> _results = [];
  bool _loading = false;
  String? _error;

  @override
  void dispose() {
    if (widget.controller == null) _ctrl.dispose();
    super.dispose();
  }

  Future<void> _search() async {
    final query = _ctrl.text.trim();
    if (query.isEmpty) return;
    setState(() {
      _loading = true;
      _error = null;
      _results = [];
    });
    try {
      final results = await widget.api.geocodeSearch(query);
      if (!mounted) return;
      setState(() => _results = results);
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _select(GeocodeResult result) {
    widget.onSelected(result);
    setState(() {
      _results = [];
      _ctrl.text = result.displayName;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(
              child: TextField(
                controller: _ctrl,
                decoration: InputDecoration(labelText: widget.hintText, border: const OutlineInputBorder(), isDense: true),
                onSubmitted: (_) => _search(),
              ),
            ),
            const SizedBox(width: 8),
            IconButton(
              onPressed: _loading ? null : _search,
              icon: _loading
                  ? const SizedBox(height: 16, width: 16, child: CircularProgressIndicator(strokeWidth: 2))
                  : const Icon(Icons.search),
            ),
          ],
        ),
        if (_error != null)
          Padding(
            padding: const EdgeInsets.only(top: 4),
            child: Text(_error!, style: const TextStyle(color: NocturneColors.error, fontSize: 12)),
          ),
        if (_results.isNotEmpty)
          ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 160),
            child: Card(
              margin: const EdgeInsets.only(top: 4),
              child: ListView(
                shrinkWrap: true,
                padding: EdgeInsets.zero,
                children: [
                  for (final r in _results)
                    ListTile(
                      dense: true,
                      leading: const Icon(Icons.place_outlined),
                      title: Text(r.displayName, maxLines: 2, overflow: TextOverflow.ellipsis),
                      onTap: () => _select(r),
                    ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}
