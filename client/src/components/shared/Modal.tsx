import React, { useEffect, useRef } from 'react';
import './Modal.css';

interface ModalProps {
    isOpen: boolean;
    onClose: () => void;
    title: string;
    children: React.ReactNode;
    footer?: React.ReactNode;
    testId?: string;
}

export const Modal: React.FC<ModalProps> = ({ isOpen, onClose, title, children, footer, testId }) => {
    const modalRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const handleEscape = (e: KeyboardEvent): void => {
            if (e.key === 'Escape') {
                onClose();
            }
        };

        if (isOpen) {
            document.addEventListener('keydown', handleEscape);
            document.body.style.overflow = 'hidden'; // Prevent background scrolling
        }

        return () => {
            document.removeEventListener('keydown', handleEscape);
            document.body.style.overflow = 'unset';
        };
    }, [isOpen, onClose]);

    if (!isOpen) return null;

    return (
        <div
            className="modal-overlay"
            data-testid={testId ? `${testId}-overlay` : undefined}
            onClick={(e) => {
            if (e.target === e.currentTarget) onClose();
            }}
        >
            <div className="modal-container" ref={modalRef} data-testid={testId}>
                <div className="modal-header" data-testid={testId ? `${testId}-header` : undefined}>
                    <h2 data-testid={testId ? `${testId}-title` : undefined}>{title}</h2>
                    <button className="modal-close" data-testid={testId ? `${testId}-close` : undefined} onClick={onClose}>&times;</button>
                </div>
                <div className="modal-content" data-testid={testId ? `${testId}-content` : undefined}>
                    {children}
                </div>
                {footer && (
                    <div className="modal-footer" data-testid={testId ? `${testId}-footer` : undefined}>
                        {footer}
                    </div>
                )}
            </div>
        </div>
    );
};
