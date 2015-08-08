var app = app || {};

(function() {
    'use strict';

    app.CanvasGraphComponent = React.createClass({
        displayName: "canvas",
        getInitialState: function() {
            return {
                blocks: app.BlockStore.getBlocks(),
                selected: [],
                shift: false,
                controlKey: false,
            }
        },
        shouldComponentUpdate: function() {
            return false;
        },
        componentDidMount: function() {
            app.BlockStore.addListener(this._onNodesUpdate);
            document.addEventListener('keydown', this._onKeyDown);
            document.addEventListener('keyup', this._onKeyUp);
        },
        componentWillUnmount: function() {
            app.BlockStore.removeListener(this._onNodesUpdate);
            document.removeEventListener('keydown', this._onKeyDown);
            document.removeEventListener('keyup', this._onKeyUp);
        },
        _onKeyDown: function(e) {
            // only fire delete if we have the stage in focus
            if (e.keyCode === 8 && e.target === document.body) {
                e.preventDefault();
                e.stopPropagation();
                this.deleteSelection();
            }

            // only fire ctrl key state if we don't have anything in focus
            if (document.activeElement === document.body && (e.keyCode === 91 || e.keyCode === 17)) {
                this.setState({
                    controlKey: true
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
                    controlKey: false
                })

            }
            if (e.shiftKey === false) this.setState({
                shift: false
            })
        },
        _onMouseDown: function(e) {
            this.setState({
                button: e.button
            })

            var ids = app.BlockStore.pickBlock(e.pageX, e.pageY);

            if (ids.length === 0) {
                app.Dispatcher.dispatch({
                    action: app.Actions.APP_DESELECT_ALL,
                });
                return
            }

            // pick the first ID
            var id = ids[0];
            if (this.state.shift === true) {
                app.Dispatcher.dispatch({
                    action: app.Actions.APP_SELECT_TOGGLE,
                    id: id
                })
            } else if (app.BlockStore.getSelected().indexOf(id) === -1) {
                app.Dispatcher.dispatch({
                    action: app.Actions.APP_SELECT,
                    id: id
                })
            }
        },
        _onMouseUp: function(e) {
            this.setState({
                button: null
            })


        },
        _onClick: function(e) {},
        _onMouseMove: function(e) {
            // if we are using our primary button, we're moving nodes and edges
            if (this.state.button === 0) {
                app.Dispatcher.dispatch({
                    action: app.Actions.APP_SELECT_MOVE,
                    dx: e.nativeEvent.movementX,
                    dy: e.nativeEvent.movementY
                })
            }
        },
        _onNodesUpdate: function() {
            var ctx = React.findDOMNode(this.refs.test).getContext('2d');
            ctx.clearRect(0, 0, this.props.width, this.props.height);
            app.BlockStore.getBlocks().forEach(function(id, i) {
                var block = app.BlockStore.getBlock(id);
                ctx.drawImage(block.canvas, block.position.x, block.position.y);
            })
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
                onDrag: this._onDrag
            }, null);
        }
    });
})();
