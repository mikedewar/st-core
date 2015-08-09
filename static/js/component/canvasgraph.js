var app = app || {};

(function() {
    'use strict';

    app.CanvasGraphComponent = React.createClass({
        displayName: "canvas",
        getInitialState: function() {
            return {
                shift: false,
                control: false,
                button: null,
                bufferNodes: document.createElement('canvas'),
                bufferSelection: document.createElement('canvas'),
                bufferStage: document.createElement('canvas'),
                mouseDownId: null,
                mouseDownX: null,
                mouseDownY: null,
                mouseLastX: null,
                mouseLastY: null,
                selecting: false,
                selection: [],
                translateX: 0, // TODO: do per-group translations, deprecate this
                translateY: 0
            }
        },
        shouldComponentUpdate: function() {
            return false;
        },
        componentDidMount: function() {
            app.BlockStore.addListener(this._onNodesUpdate);
            window.addEventListener('keydown', this._onKeyDown);
            window.addEventListener('keyup', this._onKeyUp);
            window.addEventListener('resize', this._onResize);

            this._onResize();
        },
        componentWillUnmount: function() {
            app.BlockStore.removeListener(this._onNodesUpdate);
            window.removeEventListener('keydown', this._onKeyDown);
            window.removeEventListener('keyup', this._onKeyUp);
            window.removeEventListener('resize', this._onResize);
        },
        _onResize: function(e) {
            var width = document.body.clientWidth;
            var height = document.body.clientHeight;

            // resize all the buffers
            this.state.bufferNodes.width = width;
            this.state.bufferNodes.height = height;
            this.state.bufferSelection.width = width;
            this.state.bufferSelection.height = height;
            this.state.bufferStage.width = width;
            this.state.bufferStage.height = height;

            // resize the main canvas
            React.findDOMNode(this.refs.test).width = width;
            React.findDOMNode(this.refs.test).height = height;

            // render everything again
            this._onStageUpdate();
            this._onNodesUpdate();
            this._renderBuffers();
        },
        _onKeyDown: function(e) {
            // only fire delete if we have the stage in focus
            if (e.keyCode === 8 && e.target === document.body) {
                e.preventDefault();
                e.stopPropagation();
                this.deleteSelection();
            }

            // only fire ctrl key state if we don't have anything in focus
            if (document.activeElement === document.body &&
                (e.keyCode === 91 || e.keyCode === 17)) {
                this.setState({
                    control: true
                })
            }

            if (e.shiftKey === true) {
                this.setState({
                    shift: true
                })
            }
        },
        _onKeyUp: function(e) {
            if (e.keyCode === 91 || e.keyCode === 17) {
                this.setState({
                    control: false
                })

            }
            if (e.shiftKey === false) {
                this.setState({
                    shift: false
                })
            }
        },
        _onMouseDown: function(e) {
            this.setState({
                button: e.button,
                mouseDownX: e.pageX,
                mouseDownY: e.pageY
            })

            var ids = app.BlockStore.pickBlock(e.pageX, e.pageY);

            // if we've clicked on nothing, deselect everything
            if (ids.length === 0) {
                if (this.state.shift === false) {
                    app.Dispatcher.dispatch({
                        action: app.Actions.APP_DESELECT_ALL,
                    });
                }
                this.setState({
                    mouseDownId: null
                })
                return
            }

            // pick the first ID
            var id = ids[0];
            if (this.state.shift === true) {
                app.Dispatcher.dispatch({
                    action: app.Actions.APP_SELECT_TOGGLE,
                    ids: [id]
                })
            } else if (app.BlockStore.getSelected().indexOf(id) === -1) {
                app.Dispatcher.dispatch({
                    action: app.Actions.APP_SELECT,
                    id: id
                })
            }

            this.setState({
                mouseDownId: id
            })
        },
        _onMouseUp: function(e) {
            this.setState({
                button: null
            });

            if (this.state.selecting === true) {
                this.setState({
                    selecting: false,
                    selection: []
                });
                this._selectionRectClear();
            }
        },
        _onClick: function(e) {},
        _onContextMenu: function(e) {
            e.nativeEvent.preventDefault();
        },
        _onMouseMove: function(e) {
            this.setState({
                mouseLastX: e.pageX,
                mouseLastY: e.pageY
            });

            if (this.state.button === 0 && this.state.mouseDownId !== null &&
                this.state.shift === false) {
                app.Dispatcher.dispatch({
                    action: app.Actions.APP_SELECT_MOVE,
                    dx: e.pageX - this.state.mouseLastX,
                    dy: e.pageY - this.state.mouseLastY
                })
            } else if (this.state.button === 0 && this.state.mouseDownId === null) {
                if (this.state.selected !== true) {
                    this.setState({
                        selecting: true
                    })
                }
                this._selectionRectUpdate(e.pageX, e.pageY);
            } else if (this.state.button === 2) {
                var dx = e.pageX - this.state.mouseLastX;
                var dy = e.pageY - this.state.mouseLastY;
                this.setState({
                    translateX: this.state.translateX + dx,
                    translateY: this.state.translateY + dy
                }, function() {
                    this._onStageUpdate()
                }.bind(this));
            }
        },
        _onStageUpdate: function() {
            var ctx = this.state.bufferStage.getContext('2d');
            var width = this.state.bufferStage.width;
            var height = this.state.bufferStage.height;
            var GRID_PX = 50.0;
            var translateX = this.state.translateX;
            var translateY = this.state.translateY;
            var x = translateX % GRID_PX;
            var y = translateY % GRID_PX;
            var lines = [];
            var hMax = Math.floor(width / GRID_PX);
            var vMax = Math.floor(height / GRID_PX);

            ctx.clearRect(0, 0, width, height);
            ctx.strokeStyle = 'rgb(220,220,220)';

            var grid = new Path2D();
            for (var i = 0; i <= hMax; i++) {
                grid.moveTo(x + (i * GRID_PX), 0);
                grid.lineTo(x + (i * GRID_PX), height);
            }
            for (var i = 0; i <= vMax; i++) {
                grid.moveTo(0, y + (i * GRID_PX));
                grid.lineTo(width, y + (i * GRID_PX));
            }
            ctx.stroke(grid);

            this._renderBuffers();
        },
        _selectionRectClear: function() {
            var ctx = this.state.bufferSelection.getContext('2d');
            ctx.clearRect(0, 0, this.props.width, this.props.height);

            this._renderBuffers();
        },
        _selectionRectUpdate: function(x, y) {
            var width = Math.abs(x - this.state.mouseDownX);
            var height = Math.abs(y - this.state.mouseDownY);
            var originX = Math.min(x, this.state.mouseDownX);
            var originY = Math.min(y, this.state.mouseDownY);
            var selectRect = app.BlockStore.pickArea(originX, originY, width, height);

            // get all nodes new to the selection rect
            var toggles = selectRect.filter(function(id) {
                return this.state.selection.indexOf(id) === -1
            }.bind(this))

            // get all nodes that have left the selection rect
            toggles = toggles.concat(this.state.selection.filter(function(id) {
                return selectRect.indexOf(id) === -1
            }));

            // toggle all new nodes, all nodes that have left the rect
            app.Dispatcher.dispatch({
                action: app.Actions.APP_SELECT_TOGGLE,
                ids: toggles
            })

            this.setState({
                selection: selectRect
            })

            var ctx = this.state.bufferSelection.getContext('2d');
            ctx.clearRect(0, 0, this.props.width, this.props.height);
            ctx.fillStyle = 'rgba(200,200,200,.5)';
            ctx.fillRect(originX, originY, width, height);

            this._renderBuffers();
        },
        _onNodesUpdate: function() {
            var nodesCtx = this.state.bufferNodes.getContext('2d');
            nodesCtx.clearRect(0, 0, this.props.width, this.props.height);
            app.BlockStore.getBlocks().forEach(function(id, i) {
                var block = app.BlockStore.getBlock(id);
                nodesCtx.drawImage(block.canvas, block.position.x, block.position.y);
            })

            this._renderBuffers();
        },
        _renderBuffers: function() {
            var ctx = React.findDOMNode(this.refs.test).getContext('2d');
            ctx.clearRect(0, 0, this.props.width, this.props.height);
            ctx.drawImage(this.state.bufferStage, 0, 0);
            ctx.drawImage(this.state.bufferSelection, 0, 0);
            ctx.drawImage(this.state.bufferNodes, 0, 0);
        },
        render: function() {
            return React.createElement('canvas', {
                ref: 'test',
                width: this.props.width,
                height: this.props.height,
                onMouseDown: this._onMouseDown,
                onMouseUp: this._onMouseUp,
                onDoubleClick: this.props.doubleClick,
                onClick: this._onClick,
                onMouseMove: this._onMouseMove,
                onContextMenu: this._onContextMenu,
            }, null);
        }
    });
})();
