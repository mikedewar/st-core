var app = app || {};

/* PanelListComponent
 * The right sidebar that is contains a series of panels.
 */

(function() {
    'use strict';

    app.PanelListComponent = React.createClass({
        displayName: 'PanelListComponent',
        getInitialState: function() {
            return {
                ids: app.NodeStore.getSelected(),
            }
        },
        componentDidMount: function() {
            app.NodeSelection.addListener(this._onUpdate);
        },
        componentWillUnmount: function() {
            app.NodeSelection.addListener(this._onUpdate);
        },
        _onUpdate: function() {
            this.setState({
                ids: app.NodeStore.getSelected(),
            })
        },
        render: function() {
            return React.createElement('div', {
                className: 'panel_list'
            }, this.state.ids.map(function(id) {
                return React.createElement(app.RoutesPanelComponent, {
                        key: id,
                        id: id
                    })
                    /*if (n instanceof app.Source) {
                        return React.createElement(app.ParametersPanelComponent, {
                            model: n,
                            key: id
                        }, null)
                    } else {
                        return React.createElement(app.RoutesPanelComponent, {
                            model: n,
                            key: n.data.id
                        }, null)
                    }*/
            }))
        },
    })
})();
