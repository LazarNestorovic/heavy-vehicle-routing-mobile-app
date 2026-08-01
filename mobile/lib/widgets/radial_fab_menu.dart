import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme/nocturne_theme.dart';

/// Which corner the FAB is docked in - public so callers can pin a menu to a
/// specific corner via [RadialFabMenu.initialCorner] (e.g. vehicle_list_screen
/// pins it to the left instead of the default bottom-right).
enum FabCorner { topLeft, topRight, bottomLeft, bottomRight }

class RadialFabMenuItem {
  final IconData icon;
  final String tooltip;
  final VoidCallback onTap;
  final int badgeCount;

  const RadialFabMenuItem({required this.icon, required this.tooltip, required this.onTap, this.badgeCount = 0});
}

/// Draggable radial FAB menu - a direct port of the Claude Design mock-up's
/// `arcOffset`/`CORNERS`/pointer-drag-and-snap script (`Cargo Routing App.dc.html`,
/// see documentations/features/2026-07-21-nocturne-redesign.md). By default the
/// FAB can be dragged to any of the four screen corners, snapping to the
/// nearest one on release; set [draggable] to false to pin it at
/// [initialCorner] instead (tapping still opens/closes the menu). A tap (no
/// drag) toggles an arc of sub-buttons fanned out away from whichever corner
/// it's docked in - angles are spread evenly across [items] (see _anglesFor),
/// not a fixed 5-slot table, so the spacing stays generous (no overlapping
/// sub-buttons) regardless of how many items are passed.
///
/// Meant to wrap a full-screen body (e.g. the map in ActiveTripScreen) via
/// `Stack`.
class RadialFabMenu extends StatefulWidget {
  final List<RadialFabMenuItem> items;
  final FabCorner initialCorner;
  final bool draggable;
  final IconData closedIcon;
  const RadialFabMenu({
    super.key,
    required this.items,
    this.initialCorner = FabCorner.bottomRight,
    this.draggable = true,
    this.closedIcon = Icons.add,
  });

  @override
  State<RadialFabMenu> createState() => _RadialFabMenuState();
}

class _RadialFabMenuState extends State<RadialFabMenu> {
  static const _fabSize = 58.0;
  static const _fabRadius = _fabSize / 2;
  static const _subSize = 58.0;
  static const _edgeMargin = 53.0;
  // Wide enough that even 5 sub-buttons (see _anglesFor) don't overlap -
  // a driver reported the previous radius (100) as visibly overlapping.
  static const _arcRadius = 180.0;
  static const _arcSweepStart = 95.0;
  static const _arcSweepEnd = 0.0;
  static const _dragSlop = 6.0;
  static const _snapDuration = Duration(milliseconds: 280);
  // Mirrors the mock-up's CSS `cubic-bezier(.34,1.3,.64,1)` - the y1 > 1
  // control point is what gives the snap its slight overshoot "bounce".
  static const _snapCurve = Cubic(0.34, 1.3, 0.64, 1.0);

  late FabCorner _corner;
  bool _menuOpen = false;
  bool _dragging = false;
  Offset? _dragPosition;
  Offset? _dragStart;

  @override
  void initState() {
    super.initState();
    _corner = widget.initialCorner;
  }

  // Spreads count sub-buttons evenly across the sweep [_arcSweepEnd,
  // _arcSweepStart] instead of picking from a fixed 5-slot table - keeps
  // spacing between adjacent buttons generous whether there are 3 items or 5.
  List<double> _anglesFor(int count) {
    if (count <= 0) return const [];
    if (count == 1) return const [(_arcSweepStart + _arcSweepEnd) / 2];
    final step = (_arcSweepStart - _arcSweepEnd) / (count - 1);
    return [for (var i = 0; i < count; i++) _arcSweepStart - step * i];
  }

  Offset _cornerCenter(FabCorner corner, Size size) {
    switch (corner) {
      case FabCorner.topLeft:
        return const Offset(_edgeMargin, _edgeMargin);
      case FabCorner.topRight:
        return Offset(size.width - _edgeMargin, _edgeMargin);
      case FabCorner.bottomLeft:
        return Offset(_edgeMargin, size.height - _edgeMargin);
      case FabCorner.bottomRight:
        return Offset(size.width - _edgeMargin, size.height - _edgeMargin);
    }
  }

  /// h/v sign multipliers so the sub-button arc always sweeps away from the
  /// docked corner - same table as the mock-up's `CORNERS`.
  (double, double) _arcSign(FabCorner corner) {
    switch (corner) {
      case FabCorner.bottomRight:
        return (-1, -1);
      case FabCorner.bottomLeft:
        return (1, -1);
      case FabCorner.topRight:
        return (-1, 1);
      case FabCorner.topLeft:
        return (1, 1);
    }
  }

  Offset _arcOffset(double angleDeg, double h, double v) {
    final rad = angleDeg * math.pi / 180;
    return Offset(h * _arcRadius * math.cos(rad), v * _arcRadius * math.sin(rad));
  }

  // `DragUpdateDetails.localPosition` is local to whichever RenderBox the
  // recognizer is attached to - here that's the small 58px FAB itself, which
  // moves every frame during the drag. Using it directly (as if it were
  // stable container-relative coordinates) made the drag distance nonsense.
  // Converting through the container's own RenderBox via this key keeps
  // every position in one fixed coordinate space for the whole gesture.
  final _containerKey = GlobalKey();

  Offset? _toContainerLocal(Offset global) {
    final box = _containerKey.currentContext?.findRenderObject() as RenderBox?;
    if (box == null || !box.attached) return null;
    return box.globalToLocal(global);
  }

  void _onPanStart(DragStartDetails details) {
    _dragStart = _toContainerLocal(details.globalPosition);
    _dragging = true;
  }

  void _onPanUpdate(DragUpdateDetails details, Size size) {
    if (!_dragging || _dragStart == null) return;
    final local = _toContainerLocal(details.globalPosition);
    if (local == null) return;
    final moved = (local - _dragStart!).distance > _dragSlop;
    setState(() {
      _dragPosition = Offset(
        local.dx.clamp(_fabRadius, math.max(_fabRadius, size.width - _fabRadius)),
        local.dy.clamp(_fabRadius, math.max(_fabRadius, size.height - _fabRadius)),
      );
      if (moved) _menuOpen = false;
    });
  }

  void _onPanEnd(Size size) {
    final moved = _dragPosition != null && _dragStart != null && (_dragPosition! - _dragStart!).distance > _dragSlop;
    if (moved) {
      final pos = _dragPosition!;
      final isLeft = pos.dx < size.width / 2;
      final isTop = pos.dy < size.height / 2;
      setState(() {
        _corner = isTop
            ? (isLeft ? FabCorner.topLeft : FabCorner.topRight)
            : (isLeft ? FabCorner.bottomLeft : FabCorner.bottomRight);
        _dragging = false;
        _dragPosition = null;
      });
    } else {
      setState(() {
        _dragging = false;
        _dragPosition = null;
        _menuOpen = !_menuOpen;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final size = constraints.biggest;
        final center = _dragging && _dragPosition != null ? _dragPosition! : _cornerCenter(_corner, size);
        final (h, v) = _arcSign(_corner);
        final duration = _dragging ? Duration.zero : _snapDuration;
        final angles = _anglesFor(widget.items.length);

        return Stack(
          key: _containerKey,
          children: [
            if (_menuOpen)
              Positioned.fill(
                child: GestureDetector(
                  behavior: HitTestBehavior.opaque,
                  onTap: () => setState(() => _menuOpen = false),
                ),
              ),
            for (var i = 0; i < widget.items.length; i++)
              _buildSubButton(i, widget.items[i], _arcOffset(angles[i], h, v) + center, duration),
            _buildFab(center, size, duration),
          ],
        );
      },
    );
  }

  Widget _buildSubButton(int index, RadialFabMenuItem item, Offset center, Duration duration) {
    return AnimatedPositioned(
      duration: duration,
      curve: _snapCurve,
      left: center.dx - _subSize / 2,
      top: center.dy - _subSize / 2,
      width: _subSize,
      height: _subSize,
      child: IgnorePointer(
        key: ValueKey('radial-fab-menu-ignore-$index'),
        ignoring: !_menuOpen,
        child: AnimatedScale(
          scale: _menuOpen ? 1.0 : 0.3,
          duration: duration == Duration.zero ? _snapDuration : duration,
          curve: _snapCurve,
          child: AnimatedOpacity(
            opacity: _menuOpen ? 1.0 : 0.0,
            duration: duration == Duration.zero ? _snapDuration : duration,
            child: Material(
              color: NocturneColors.surface,
              shape: const CircleBorder(side: BorderSide(color: NocturneColors.accent700)),
              elevation: 4,
              child: InkWell(
                customBorder: const CircleBorder(),
                onTap: () {
                  setState(() => _menuOpen = false);
                  item.onTap();
                },
                child: Tooltip(
                  message: item.tooltip,
                  child: Stack(
                    alignment: Alignment.center,
                    children: [
                      Icon(item.icon, color: NocturneColors.accent300, size: 20),
                      if (item.badgeCount > 0)
                        Positioned(
                          top: 2,
                          right: 2,
                          child: Container(
                            padding: const EdgeInsets.symmetric(horizontal: 3),
                            constraints: const BoxConstraints(minWidth: 16, minHeight: 16),
                            decoration: const BoxDecoration(color: NocturneColors.accent, shape: BoxShape.circle),
                            alignment: Alignment.center,
                            child: Text(
                              '${item.badgeCount}',
                              style:
                                  const TextStyle(fontSize: 10, fontWeight: FontWeight.w600, color: NocturneColors.bg),
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildFab(Offset center, Size size, Duration duration) {
    final fab = Material(
      color: NocturneColors.surface,
      shape: const CircleBorder(side: BorderSide(color: NocturneColors.accent)),
      elevation: 8,
      child: Center(
        child: AnimatedSwitcher(
          duration: const Duration(milliseconds: 150),
          child: Icon(
            _menuOpen ? Icons.close : widget.closedIcon,
            key: ValueKey(_menuOpen),
            color: NocturneColors.accent,
            size: 24,
          ),
        ),
      ),
    );

    return AnimatedPositioned(
      duration: duration,
      curve: _snapCurve,
      left: center.dx - _fabRadius,
      top: center.dy - _fabRadius,
      width: _fabSize,
      height: _fabSize,
      child: widget.draggable
          ? GestureDetector(
              onPanStart: _onPanStart,
              onPanUpdate: (details) => _onPanUpdate(details, size),
              onPanEnd: (_) => _onPanEnd(size),
              child: fab,
            )
          : GestureDetector(
              onTap: () => setState(() => _menuOpen = !_menuOpen),
              child: fab,
            ),
    );
  }
}
