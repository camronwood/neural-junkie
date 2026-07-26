/// <reference types="react" />
declare module 'react' {
  function createElement<P extends {}>(type: any, props?: P, children?: React.JSX.Element[]): React.ReactElement;
  namespace JSX {
    interface Element extends React.ReactElement {}
    interface ElementAttributesProperty {
      element: never;
    }
    interface IntrinsicElements {
      [elem: string]: any;
    }
  }
}
declare module 'react/jsx-runtime' {
  export function createElement(type: any, props?: any): any;
  export function cloneElement(el: React.ReactElement<any, any>, props?: any, children?: any): React.ReactElement<any, any>;
}