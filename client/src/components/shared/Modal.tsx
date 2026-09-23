import React, { useEffect, useId, useRef } from 'react';
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
    const modalRef = useRef<HTMLDialogElement>(null);
    const titleId = useId();

    useEffect(() => {
        const dialog = modalRef.current;
        if (!isOpen || !dialog) return;
        const previousFocus = document.activeElement;
        const previousOverflow = document.body.style.overflow;
        dialog.showModal();
        document.body.style.overflow = 'hidden';
        return () => {
            dialog.close();
            document.body.style.overflow = previousOverflow;
            if (previousFocus instanceof HTMLElement && previousFocus.isConnected) previousFocus.focus();
        };
    }, [isOpen]);

    if (!isOpen) return null;

    return (
        <dialog
            ref={modalRef}
            aria-labelledby={titleId}
            aria-modal="true"
            onCancel={(event) => { event.preventDefault(); onClose(); }}
            onKeyDown={(event) => {
                if (event.key !== 'Tab') return;
                const controls = Array.from(event.currentTarget.querySelectorAll<HTMLElement>(
                    'button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href], [tabindex="0"]',
                )).filter(element => element.getClientRects().length > 0);
                const first = controls[0];
                const last = controls.at(-1);
                if (event.shiftKey && document.activeElement === first) {
                    event.preventDefault(); last?.focus();
                } else if (!event.shiftKey && document.activeElement === last) {
                    event.preventDefault(); first?.focus();
                }
            }}
            className="modal-overlay"
            data-testid={testId ? `${testId}-overlay` : undefined}
            onClick={(e) => {
            if (e.target === e.currentTarget) onClose();
            }}
        >
            <div className="modal-container" data-testid={testId}>
                <div className="modal-header" data-testid={testId ? `${testId}-header` : undefined}>
                    <h2 id={titleId} data-testid={testId ? `${testId}-title` : undefined}>{title}</h2>
                    <button type="button" aria-label={`Close ${title}`} className="modal-close" data-testid={testId ? `${testId}-close` : undefined} onClick={onClose}>&times;</button>
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
        </dialog>
    );
};
